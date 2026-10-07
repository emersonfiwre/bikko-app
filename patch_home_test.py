import re

with open('internal/domain/home_test.go', 'r') as f:
    content = f.read()

content = content.replace('Items: []Category{', 'Items: []HomeItem{')
content = content.replace('{ID: "c1", Name: "Pintura"}', '{ID: "c1", Title: "Pintura"}')

with open('internal/domain/home_test.go', 'w') as f:
    f.write(content)
