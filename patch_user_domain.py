import re

with open('internal/domain/user.go', 'r') as f:
    content = f.read()

content = re.sub(r'\s*PasswordHash\s+string\s+`json:"-"`', '', content)
content = re.sub(r'CreateUser\(ctx context\.Context, user \*User, password string\) \(\*User, error\)', 'CreateUser(ctx context.Context, user *User) (*User, error)', content)

with open('internal/domain/user.go', 'w') as f:
    f.write(content)

with open('internal/domain/user_test.go', 'r') as f:
    content = f.read()

content = re.sub(r'\s*PasswordHash:\s*"[^"]*",', '', content)
content = re.sub(r'(?s)// PasswordHash must be omitted.*?\n\t}', '', content)
content = re.sub(r'(?s)if unmarshaled\.PasswordHash != "".*?\n\t}', '', content)

with open('internal/domain/user_test.go', 'w') as f:
    f.write(content)
