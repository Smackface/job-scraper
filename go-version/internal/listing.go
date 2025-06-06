package internal_linkedin_scraper

import (
	"log"

	"github.com/sashabaranov/go-openai/jsonschema"
)

type Listing struct {
	Title        string `json:"title" jsonschema_description:"The job title"`
	Location     string `json:"location" jsonschema_description:"Geographic location of the job"`
	Company      string `json:"company" jsonschema_description:"Company offering the position"`
	Pay          string `json:"pay" jsonschema_description:"Compensation details for the role"`
	Technologies string `json:"technologies" jsonschema_description:"Technologies required for the job"`
	Seniority    string `json:"seniority" jsonschema:"enum=junior,enum=mid,enum=senior" jsonschema_description:"Experience level required"`
	Description  string `json:"description" jsonschema_description:"Detailed job description"`
	Contact      string `json:"contact" jsonschema_description:"Contact information, like email, phone, website"`
}

func generateSchema[T any]() *jsonschema.Definition {
	var listing T
	schema, err := jsonschema.GenerateSchemaForType(listing)
	if err != nil {
		log.Fatalf("GenerateSchemaForType error: %v", err)
	}

	return schema
}

func GetListingSchema() *jsonschema.Definition {
	return generateSchema[Listing]()
}
