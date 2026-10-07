import re

with open('internal/service/auth_service.go', 'r') as f:
    content = f.read()

# Remove ForgotPassword
content = re.sub(r'\s*ForgotPassword\(ctx context\.Context, email string\) error', '', content)
content = re.sub(r'func \(s \*authService\) ForgotPassword\(.*?\n}\n', '', content, flags=re.DOTALL)

# In Register, remove hashedPassword logic and pass only user
content = re.sub(r'\s*var hashedPassword string.*?hashedPassword = "\$2a\$10\$BikkoPhoneAuthVerifiedUserPlaceholderHash"\n\t}', '', content, flags=re.DOTALL)
content = content.replace('createdUser, err := s.repo.CreateUser(ctx, user, hashedPassword)', 'createdUser, err := s.repo.CreateUser(ctx, user)')

# In Login, remove Fallback to Email / Password login
login_start = content.find('// Fallback to Email / Password login')
if login_start != -1:
    login_end = content.find('return &model.AuthResponse', login_start)
    if login_end != -1:
        login_end = content.find('}', login_end) + 1
        content = content[:login_start] + 'return nil, ErrInvalidCredentials\n}\n'

with open('internal/service/auth_service.go', 'w') as f:
    f.write(content)

