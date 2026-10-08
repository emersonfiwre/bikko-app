import sys

content = open("internal/usecase/home_usecase.go", "r").read()
old = """func (uc *HomeUseCase) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	return uc.categoryRepo.GetByID(ctx, id)
}"""

new = """func (uc *HomeUseCase) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	cat, err := uc.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	services, err := uc.serviceRepo.GetServicesByCategoryID(ctx, id)
	if err == nil {
		cat.Services = services
	}
	return cat, nil
}"""

if old in content:
    content = content.replace(old, new)
    open("internal/usecase/home_usecase.go", "w").write(content)
