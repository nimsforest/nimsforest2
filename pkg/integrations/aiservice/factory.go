package aiservice

import (
	"fmt"
	"sync"

	pkgaiservice "github.com/yourusername/nimsforest/pkg/infrastructure/aiservice"
)

// ServiceType represents the type of AI service to create.
type ServiceType string

// Constants for supported AI service types.
const (
	ServiceTypeClaude ServiceType = "claude"
	ServiceTypeOpenAI ServiceType = "openai"
	ServiceTypeGemini ServiceType = "gemini"
	// ServiceTypeXAI    ServiceType = "xai" // Add if/when supported
)

// ServiceFactoryFunc defines the signature for functions that can create an AIService.
type ServiceFactoryFunc func(apiKey, model string) (pkgaiservice.AIService, error)

// serviceFactories holds the registered factory functions for each service type.
// We use a sync.Map for safe concurrent access, although registration typically
// happens only during init phases.
var serviceFactories sync.Map // map[ServiceType]ServiceFactoryFunc

// RegisterService allows internal implementations to register their factory function.
func RegisterService(serviceType ServiceType, factory ServiceFactoryFunc) {
	// Store the factory function in the sync.Map.
	// Using sync.Map avoids potential race conditions if registration were concurrent,
	// though init() functions run sequentially per package.
	if _, loaded := serviceFactories.LoadOrStore(serviceType, factory); loaded {
		// This prevents accidental duplicate registration, adjust if overwriting is desired.
		panic(fmt.Sprintf("AIService factory already registered for type: %s", serviceType))
	}
}

// NewService creates and initializes a new AI service based on the specified type
// by looking up and calling the registered factory function.
func NewService(serviceType ServiceType, apiKey, model string) (pkgaiservice.AIService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required to create an AI service")
	}
	if model == "" {
		return nil, fmt.Errorf("model name is required to create an AI service")
	}

	// Load the factory function from the sync.Map.
	factoryInterface, ok := serviceFactories.Load(serviceType)
	if !ok {
		return nil, fmt.Errorf("unsupported or unregistered AI service type: %s", serviceType)
	}

	// Assert the type of the loaded value to our factory function type.
	factory, ok := factoryInterface.(ServiceFactoryFunc)
	if !ok {
		// This should not happen if registration is done correctly.
		return nil, fmt.Errorf("invalid factory function registered for type: %s", serviceType)
	}

	// Call the registered factory function.
	return factory(apiKey, model)
}
