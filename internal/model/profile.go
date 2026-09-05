package model

type ProfileResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Rating       string `json:"rating"`
	TotalRatings int    `json:"totalRatings"`
}
