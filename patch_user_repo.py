import re

with open('internal/infrastructure/persistence/postgres/user_repository.go', 'r') as f:
    content = f.read()

content = content.replace('func (r *userRepository) CreateUser(ctx context.Context, user *domain.User, password string) (*domain.User, error)', 'func (r *userRepository) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)')
content = re.sub(r'// \'password\' argument is the hashed password.*?\n', '', content)
content = re.sub(r'// Wait, interface domain.*?\n', '', content)
content = re.sub(r'// I\'ll expect the password passed here.*?\n', '', content)
content = re.sub(r'// Or I can just hash it.*?\n', '', content)
content = re.sub(r'// Let\'s assume the service hashes it.*?\n', '', content)
content = content.replace('INSERT INTO users (full_name, email, phone, cpf, password_hash)', 'INSERT INTO users (full_name, email, phone, cpf)')
content = content.replace('VALUES ($1, $2, $3, $4, $5)', 'VALUES ($1, $2, $3, $4)')
content = content.replace('password,\n\t\t)', ')')
content = content.replace('SELECT id, full_name, email, COALESCE(phone, \'\'), COALESCE(cpf, \'\'), password_hash, rating, total_ratings, is_bikker, COALESCE(device_token, \'\'), created_at, updated_at', 'SELECT id, full_name, email, COALESCE(phone, \'\'), COALESCE(cpf, \'\'), rating, total_ratings, is_bikker, COALESCE(device_token, \'\'), created_at, updated_at')
content = content.replace('\t\t&user.PasswordHash,\n', '')

with open('internal/infrastructure/persistence/postgres/user_repository.go', 'w') as f:
    f.write(content)
