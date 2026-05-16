package model

type MealPlanDay struct {
	Day       string  `json:"day"`
	Breakfast Recipe  `json:"breakfast"`
	Lunch     Recipe  `json:"lunch"`
	Dinner    Recipe  `json:"dinner"`
	Snack     *Recipe `json:"snack,omitempty"`
}

type MealPlan struct {
	Days        []MealPlanDay `json:"days"`
	Tips        []string      `json:"tips"`
	GroceryList []string      `json:"grocery_list"`
}
