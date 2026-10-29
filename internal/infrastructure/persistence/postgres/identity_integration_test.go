package postgres

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"bikko-app/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func identityFixture(t *testing.T, pool *pgxpool.Pool) *domain.User {
	t.Helper()
	repo := NewUserRepository(pool)
	user, err := repo.CreateUser(context.Background(), &domain.User{FullName: "Identity integration", Email: uuid.NewString() + "@test.invalid", Phone: fmt.Sprintf("119%08d", rand.IntN(100000000))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, user.ID); err != nil {
			t.Errorf("cleanup identity: %v", err)
		}
	})
	return user
}
func TestIdentityRegistrationAndUpgradeRollback(t *testing.T) {
	pool := contractTestPool(t)
	ctx := context.Background()
	repo := NewUserRepository(pool)
	email := uuid.NewString() + "@test.invalid"
	_, err := repo.CreateUser(ctx, &domain.User{FullName: "Rollback", Email: email, Phone: fmt.Sprintf("119%08d", rand.IntN(100000000)), IsBikker: true, Bikker: &domain.Bikker{Profession: strings.Repeat("x", 101), IsActive: true}})
	if err == nil {
		t.Fatal("invalid provider registration succeeded")
	}
	if _, err := repo.GetByEmail(ctx, email); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("partial identity retained: %v", err)
	}
	user := identityFixture(t, pool)
	_, err = repo.UpgradeToBikker(ctx, user.ID, &domain.Bikker{Profession: strings.Repeat("x", 101), IsActive: true})
	if err == nil {
		t.Fatal("invalid provider upgrade succeeded")
	}
	unchanged, err := repo.GetByID(ctx, user.ID)
	if err != nil || unchanged.IsBikker {
		t.Fatalf("partial upgrade retained: %+v %v", unchanged, err)
	}
}
func TestIdentityPhoneUnicityAndRealClientLocation(t *testing.T) {
	pool := contractTestPool(t)
	ctx := context.Background()
	repo := NewUserRepository(pool)
	user := identityFixture(t, pool)
	_, err := repo.CreateUser(ctx, &domain.User{FullName: "Duplicate", Email: uuid.NewString() + "@test.invalid", Phone: "+55" + user.Phone})
	if !errors.Is(err, domain.ErrConflictPhone) || !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("same canonical phone accepted: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE users SET location_name='Pinheiros',location_geom=ST_SetSRID(ST_MakePoint(-46.6823,-23.5641),4326)::geography WHERE id=$1`, user.ID); err != nil {
		t.Fatal(err)
	}
	actual, err := repo.GetByPhone(ctx, "+55"+user.Phone)
	if err != nil || actual.Latitude == nil || actual.Longitude == nil || *actual.Latitude != -23.5641 || *actual.Longitude != -46.6823 {
		t.Fatal(actual, err)
	}
	if _, err := repo.GetByPhone(ctx, user.Phone[2:]); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("suffix lookup accepted: %v", err)
	}
}
func TestIdentityDevicesIndependentAndAccountDeletionAtomic(t *testing.T) {
	pool := contractTestPool(t)
	ctx := context.Background()
	repo := NewUserRepository(pool).(*userRepository)
	user := identityFixture(t, pool)
	b := &domain.Bikker{Profession: "Electrician", Latitude: -23.5614, Longitude: -46.6914, IsActive: true}
	if _, err := repo.UpgradeToBikker(ctx, user.ID, b); err != nil {
		t.Fatal(err)
	}
	category, service := uuid.NewString(), uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO categories(id,name,is_active) VALUES($1,$2,TRUE)`, category, category); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM categories WHERE id=$1`, category); err != nil {
			t.Error(err)
		}
	})
	if _, err := pool.Exec(ctx, `INSERT INTO services(id,bikker_id,category_id,name,is_active) VALUES($1,$2,$3,'Test service',TRUE)`, service, user.ID, category); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM services WHERE id=$1`, service); err != nil {
			t.Error(err)
		}
	})
	clientToken, providerToken := uuid.NewString(), uuid.NewString()
	if err := repo.RegisterDevice(ctx, user.ID, "client", clientToken); err != nil {
		t.Fatal(err)
	}
	if err := repo.RegisterDevice(ctx, user.ID, "provider", providerToken); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteDevice(ctx, user.ID, "client", clientToken); err != nil {
		t.Fatal(err)
	}
	tokens, err := repo.GetDeviceTokens(ctx, user.ID)
	if err != nil || len(tokens) != 1 || tokens[0] != providerToken {
		t.Fatal(tokens, err)
	}
	if err := repo.DeleteUser(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(ctx, user.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted account readable: %v", err)
	}
	tokens, err = repo.GetDeviceTokens(ctx, user.ID)
	if err != nil || len(tokens) != 0 {
		t.Fatal(tokens, err)
	}
	var active bool
	if err := pool.QueryRow(ctx, `SELECT is_active FROM services WHERE id=$1`, service).Scan(&active); err != nil || active {
		t.Fatalf("deleted provider still selling: %v %v", active, err)
	}
	if err := pool.QueryRow(ctx, `SELECT is_active FROM bikkers WHERE id=$1`, user.ID).Scan(&active); err != nil || active {
		t.Fatalf("deleted provider still active: %v %v", active, err)
	}
}
