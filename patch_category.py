import sys

content = open("internal/domain/category.go", "r").read()
if "Services []Service" not in content:
    content = content.replace("CreatedAt   time.Time `json:\"created_at,omitempty\"`\n}", "CreatedAt   time.Time `json:\"created_at,omitempty\"`\n\tServices    []Service `json:\"services,omitempty\"`\n}")
    open("internal/domain/category.go", "w").write(content)
