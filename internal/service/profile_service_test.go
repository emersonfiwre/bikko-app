package service

import (
	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/model"
	"context"
	"errors"
	"testing"
)

type fakeProfileRepo struct {
	profileResp *model.ProfileResponse
	profileErr  error
	deleteErr   error
}

func (f *fakeProfileRepo) GetProfileByID(context.Context, string) (*model.ProfileResponse, error) {
	return f.profileResp, f.profileErr
}
func (f *fakeProfileRepo) DeleteAccount(context.Context, string) error { return f.deleteErr }

func TestProfileRejectsMissingIdentityAndNeverFallsBack(t *testing.T) {
	repo := newMockAuthUserRepo()
	svc := NewProfileService(repo, &fakeProfileRepo{profileResp: &model.ProfileResponse{Name: "Fixture"}})
	for _, id := range []string{"", "missing"} {
		if _, err := svc.GetProfile(context.Background(), id); err == nil {
			t.Fatalf("accepted identity %q", id)
		}
	}
}
func TestProfileExposesOnlyActualFieldsAndScopedOrders(t *testing.T) {
	repo := newMockAuthUserRepo()
	repo.users["11999990000"] = &domain.User{ID: "lucas", FullName: "Lucas Ferreira Mendes", Phone: "11999990000"}
	svc := NewProfileService(repo, &fakeProfileRepo{profileResp: &model.ProfileResponse{Orders: []model.ProfileOrderItem{}}})
	profile, err := svc.GetProfile(context.Background(), "lucas")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != "Lucas Ferreira Mendes" || profile.Profession != "" || profile.Latitude != 0 || len(profile.Orders) != 0 || profile.IsActive != nil {
		t.Fatalf("fabricated profile: %+v", profile)
	}
}
func TestProfileOrdersFailureIsNotAnEmptySuccess(t *testing.T) {
	repo := newMockAuthUserRepo()
	repo.users["11999990000"] = &domain.User{ID: "lucas", Phone: "11999990000"}
	want := errors.New("database unavailable")
	svc := NewProfileService(repo, &fakeProfileRepo{profileErr: want})
	if _, err := svc.GetProfile(context.Background(), "lucas"); !errors.Is(err, want) {
		t.Fatal(err)
	}
}
func TestPhoneChangeRequiresProofOfRequestedNumber(t *testing.T) {
	for _, tc := range []struct {
		name, token, claim string
		want               error
	}{{"no token", "", "+5511988880000", ErrPhoneChangeRequiresFirebaseToken}, {"different claim", "valid", "+5511977770000", ErrInvalidFirebaseToken}, {"valid", "valid", "+5511988880000", nil}} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockAuthUserRepo()
			repo.users["11999990000"] = &domain.User{ID: "lucas", Phone: "11999990000"}
			jwt := security.NewJWTService("test-secret")
			auth := NewAuthService(repo, jwt, verifiedPhoneToken{phone: tc.claim})
			svc := NewProfileService(repo, nil, WithAuthAndJWT(auth, jwt))
			token, err := svc.UpdateProfileWithToken(context.Background(), "lucas", "Lucas", "lucas@test.com", "11988880000", tc.token)
			if tc.want != nil {
				if !errors.Is(err, tc.want) {
					t.Fatal(err)
				}
				if _, exists := repo.users["11999990000"]; !exists {
					t.Fatal("invalid proof changed phone")
				}
			} else {
				if err != nil || token == "" {
					t.Fatal(token, err)
				}
			}
		})
	}
}
func TestSamePhoneDoesNotRequireOtpAndBlankDoesNotClearIt(t *testing.T) {
	repo := newMockAuthUserRepo()
	repo.users["11999990000"] = &domain.User{ID: "lucas", Phone: "11999990000"}
	svc := NewProfileService(repo, nil)
	for _, phone := range []string{"+55 (11) 99999-0000", ""} {
		if _, err := svc.UpdateProfileWithToken(context.Background(), "lucas", "Lucas", "lucas@test.com", phone, ""); err != nil {
			t.Fatal(err)
		}
	}
	if repo.users["11999990000"] == nil {
		t.Fatal("phone was cleared")
	}
}
func TestPhoneAlreadyAssignedCannotBeTakenOver(t *testing.T) {
	repo := newMockAuthUserRepo()
	repo.users["11999990000"] = &domain.User{ID: "lucas", Phone: "11999990000"}
	repo.users["11988880000"] = &domain.User{ID: "rafael", Phone: "11988880000"}
	jwt := security.NewJWTService("test-secret")
	auth := NewAuthService(repo, jwt, verifiedPhoneToken{phone: "+5511988880000"})
	svc := NewProfileService(repo, nil, WithAuthAndJWT(auth, jwt))
	if _, err := svc.UpdateProfileWithToken(context.Background(), "lucas", "Lucas", "lucas@test.com", "11988880000", "valid"); !errors.Is(err, ErrPhoneAlreadyInUse) {
		t.Fatal(err)
	}
}
