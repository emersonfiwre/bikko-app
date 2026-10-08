import re

with open('internal/usecase/home_usecase.go', 'r') as f:
    content = f.read()

mapper_code = """
	collections := make([]domain.HomeCollection, 0)

	if len(categories) > 0 {
		var items []domain.HomeItem
		for _, c := range categories {
			items = append(items, domain.HomeItem{
				ID:       c.ID,
				Name:     c.Name,
				Title:    c.Name,
				IconURL:  c.IconURL,
				Subtitle: c.Description,
			})
		}
		collections = append(collections, domain.HomeCollection{
			ID:    "categories_collection",
			Name:  "Categorias",
			Type:  "Category",
			Items: items,
		})
	}

	if len(adviceServices) > 0 {
		var items []domain.HomeItem
		for _, s := range adviceServices {
			items = append(items, domain.HomeItem{
				ID:           s.ID,
				Title:        s.Name,
				Subtitle:     s.Description,
				ImageURL:     s.ThumbnailURL,
				Rating:       s.ReviewsAverage,
				ProviderName: s.ProviderName,
				CategoryID:   s.CategoryID,
				BikkerID:     s.BikkerID,
				Latitude:     s.Latitude,
				Longitude:    s.Longitude,
			})
		}
		collections = append(collections, domain.HomeCollection{
			ID:    "advice_collection",
			Name:  "Recomendações para Você",
			Type:  "Advice",
			Items: items,
		})
	}

	if len(nearBikkers) > 0 {
		var items []domain.HomeItem
		for _, nb := range nearBikkers {
			items = append(items, domain.HomeItem{
				ID:       nb.ID,
				Title:    nb.Title,
				Subtitle: nb.Subtitle,
				ImageURL: nb.ImageURL,
				Distance: nb.Distance,
			})
		}
		collections = append(collections, domain.HomeCollection{
			ID:    "near_collection",
			Name:  "Profissionais por Perto",
			Type:  "ProfileNear",
			Items: items,
		})
	}
"""

# Replace the collections part
start_idx = content.find('collections := make([]domain.HomeCollection, 0)')
end_idx = content.find('return &domain.HomeResponse{', start_idx)

new_content = content[:start_idx] + mapper_code.strip() + '\n\n\t' + content[end_idx:]

with open('internal/usecase/home_usecase.go', 'w') as f:
    f.write(new_content)
