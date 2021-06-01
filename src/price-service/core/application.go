package core

import (
	"database/sql"
	"fmt"
)

type PriceApplication struct {
	priceRepository *PriceRepository
}

func NewPriceApplication(db *sql.DB) *PriceApplication {
	var priceRepository = &PriceRepository{DB: db}

	return &PriceApplication{
		priceRepository: priceRepository,
	}
}

func (app *PriceApplication) GetAllPrices(filters *PriceFilters) ([]Price, error) {
	return app.priceRepository.GetPricesByFilter(filters)
}

func (app *PriceApplication) ProcessEvent(id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	// TODO implement
	/*switch data.(type) {
	case event.ProductCreated:
		app.createProduct(data.(event.ProductCreated))
		break
	case event.ProductArchived:
		app.createUser(data.(event.UserCreated))
		break
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}*/
}

type PriceFilters struct {
	ProductIds []string `json:"productIds" binding:"required"`
}
