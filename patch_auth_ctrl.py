import re

with open('internal/controller/auth_controller.go', 'r') as f:
    content = f.read()

content = re.sub(r'type ForgotPasswordRequest struct {.*?\n}', '', content, flags=re.DOTALL)
content = re.sub(r'func \(ctrl \*AuthController\) ForgotPassword\(.*?\n}\n', '', content, flags=re.DOTALL)

with open('internal/controller/auth_controller.go', 'w') as f:
    f.write(content)

with open('internal/router/router.go', 'r') as f:
    content = f.read()

content = content.replace('auth.POST("/forgot-password", authCtrl.ForgotPassword)', '')

with open('internal/router/router.go', 'w') as f:
    f.write(content)

