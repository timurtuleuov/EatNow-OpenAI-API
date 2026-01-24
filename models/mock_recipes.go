package model

func MockRecipes() []Recipe {
	return []Recipe{
		{
			Title:            "Спагетти Карбонара",
			Description:      ptr("Классическая итальянская паста с яйцами, сыром, беконом и перцем."),
			Servings:         2,
			TotalTimeMinutes: 25,
			Difficulty:       ptrDiff(DifficultyMedium),
			Ingredients: []Ingredient{
				{ID: "i1", Name: "Спагетти", Quantity: ptrFloat(200), Unit: ptr("г")},
				{ID: "i2", Name: "Бекон", Quantity: ptrFloat(100), Unit: ptr("г")},
				{ID: "i3", Name: "Яйца", Quantity: ptrFloat(2), Unit: ptr("шт")},
				{ID: "i4", Name: "Пармезан", Quantity: ptrFloat(50), Unit: ptr("г")},
				{ID: "i5", Name: "Чёрный перец"},
			},
			Steps: []StepModel{
				{Order: 1, Description: "Отвари пасту до состояния аль денте."},
				{Order: 2, Description: "Обжарь бекон до хрустящей корочки."},
				{Order: 3, Description: "Смешай яйца и сыр в миске."},
				{Order: 4, Description: "Соедини всё вместе и приправь перцем."},
			},
			Tags:      []string{"итальянская", "паста", "быстро"},
			ImageURL:  ptr("https://example.com/images/carbonara.jpg"),
			Source:    ptr("local"),
			Nutrition: map[string]interface{}{"калории": 520, "белки": "22г"},
		},
		{
			Title:            "Куриное карри",
			Description:      ptr("Пряное карри с курицей и кокосовым молоком в индийском стиле."),
			Servings:         4,
			TotalTimeMinutes: 45,
			Difficulty:       ptrDiff(DifficultyMedium),
			Ingredients: []Ingredient{
				{ID: "i6", Name: "Куриная грудка", Quantity: ptrFloat(500), Unit: ptr("г")},
				{ID: "i7", Name: "Лук", Quantity: ptrFloat(1), Unit: ptr("шт")},
				{ID: "i8", Name: "Чеснок", Quantity: ptrFloat(3), Unit: ptr("зубчика")},
				{ID: "i9", Name: "Паста карри", Quantity: ptrFloat(2), Unit: ptr("ст. ложки")},
				{ID: "i10", Name: "Кокосовое молоко", Quantity: ptrFloat(400), Unit: ptr("мл")},
			},
			Steps: []StepModel{
				{Order: 1, Description: "Обжарь лук и чеснок до золотистого цвета."},
				{Order: 2, Description: "Добавь курицу и немного обжарь."},
				{Order: 3, Description: "Вмешай пасту карри и кокосовое молоко."},
				{Order: 4, Description: "Туши до готовности курицы."},
			},
			Tags:     []string{"индийская", "острое", "курица"},
			ImageURL: ptr("https://example.com/images/chicken_curry.jpg"),
			Source:   ptr("remote"),
		},
		{

			Title:            "Авокадо-тост",
			Description:      ptr("Быстрый и полезный завтрак с авокадо и яйцом."),
			Servings:         1,
			TotalTimeMinutes: 10,
			Difficulty:       ptrDiff(DifficultyEasy),
			Ingredients: []Ingredient{
				{ID: "i11", Name: "Авокадо", Quantity: ptrFloat(1)},
				{ID: "i12", Name: "Хлеб", Quantity: ptrFloat(2), Unit: ptr("ломтика")},
				{ID: "i13", Name: "Яйцо", Quantity: ptrFloat(1), Unit: ptr("шт")},
				{ID: "i14", Name: "Соль"},
				{ID: "i15", Name: "Лимонный сок"},
			},
			Steps: []StepModel{
				{Order: 1, Description: "Поджарь хлеб до хрустящей корочки."},
				{Order: 2, Description: "Разомни авокадо с лимонным соком и солью."},
				{Order: 3, Description: "Намажь на хлеб и добавь сверху жареное яйцо."},
			},
			Tags:     []string{"завтрак", "здоровое"},
			ImageURL: ptr("https://example.com/images/avocado_toast.jpg"),
			Source:   ptr("local"),
		},
	}
}

// Вспомогательные функции
func ptr(s string) *string             { return &s }
func ptrFloat(f float64) *float64      { return &f }
func ptrDiff(d Difficulty) *Difficulty { return &d }
