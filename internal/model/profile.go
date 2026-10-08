package model

type ProfileOrderCategory struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"icon_url"`
}

type ProfileOrderItem struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Photos         []string              `json:"photos"`
	Thumbnail      string                `json:"thumbnail"`
	Description    string                `json:"description"`
	Distance       string                `json:"distance"`
	ReviewsAverage string                `json:"reviews_average"`
	TotalReviews   int                   `json:"total_reviews"`
	BikkerID       string                `json:"bikker_id"`
	Category       *ProfileOrderCategory `json:"category,omitempty"`
}

type ProfileResponse struct {
	ID           string             `json:"id"`
	ImageProfile string             `json:"image_profile"`
	Name         string             `json:"name"`
	Email        string             `json:"email"`
	Phone        string             `json:"phone"`
	Rating       string             `json:"rating"`
	TotalRatings int                `json:"total_ratings"`
	Orders       []ProfileOrderItem `json:"orders"`
}
