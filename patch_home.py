import re

with open('internal/domain/home.go', 'r') as f:
    content = f.read()

home_item_struct = """
type HomeItem struct {
	ID          string  `json:"id"`
	Title       string  `json:"title,omitempty"`
	Subtitle    string  `json:"subtitle,omitempty"`
	ImageURL    string  `json:"image_url,omitempty"`
	IconURL     string  `json:"icon_url,omitempty"`
	Name        string  `json:"name,omitempty"`
	Distance    string  `json:"distance,omitempty"`
	Rating      float64 `json:"rating,omitempty"`
	PriceAmount float64 `json:"price_amount,omitempty"`
	Badge       string  `json:"badge,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	ProviderName string `json:"provider_name,omitempty"`
	CategoryID  string  `json:"category_id,omitempty"`
	BikkerID    string  `json:"bikker_id,omitempty"`
}

type HomeCollection struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Type  string     `json:"type"`
	Items []HomeItem `json:"items"`
}
"""

content = re.sub(r'type HomeCollection struct \{.*?\n}', home_item_struct.strip(), content, flags=re.DOTALL)

with open('internal/domain/home.go', 'w') as f:
    f.write(content)

