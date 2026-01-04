package brain

import (
	"fmt"
	"sync"
	"time"

	"errors"

	pkgaiservice "github.com/yourusername/nimsforest/pkg/infrastructure/aiservice"
	aifactory "github.com/yourusername/nimsforest/pkg/integrations/aiservice"
)

// Knowledge represents a piece of information stored in the brain
type Knowledge struct {
	ID        string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	Tags      []string
}

// --- Brain Registration ---

// BrainType represents the type of brain implementation to create.
type BrainType string

// Constants for supported Brain types.
const (
	BrainTypeHitchhiker BrainType = "hitchhiker"
	BrainTypeFixed      BrainType = "fixed"
	BrainTypeGenerative BrainType = "generative"
	BrainTypeClaudeCLI  BrainType = "claudecli"
	BrainTypeLlamaCPP   BrainType = "llamacpp"
)

// SimpleBrainFactoryFunc defines the signature for factories that create simple brains (no dependencies).
type SimpleBrainFactoryFunc func() (Brain, error)

// SmartBrainFactoryFunc defines the signature for functions that can create Brain instances
// which potentially require an AIService for their operation.
// The factory function receives the AIService potentially configured externally.
type SmartBrainFactoryFunc func(service pkgaiservice.AIService) (Brain, error)

// brainFactories holds the registered factory functions for each brain type.
// Using two maps to handle different factory function signatures.
var (
	simpleBrainFactories sync.Map // map[BrainType]SimpleBrainFactoryFunc
	smartBrainFactories  sync.Map // map[BrainType]SmartBrainFactoryFunc
)

// RegisterSimpleBrain allows internal hitchhiker/fixed brain implementations to register their factory.
func RegisterSimpleBrain(brainType BrainType, factory SimpleBrainFactoryFunc) {
	if _, loaded := simpleBrainFactories.LoadOrStore(brainType, factory); loaded {
		panic(fmt.Sprintf("Hitchhiker/Fixed Brain factory already registered for type: %s", brainType))
	}
}

// RegisterSmartBrain allows internal generative brain implementations to register their factory.
func RegisterSmartBrain(brainType BrainType, factory SmartBrainFactoryFunc) {
	if _, loaded := smartBrainFactories.LoadOrStore(brainType, factory); loaded {
		panic(fmt.Sprintf("Generative Brain factory already registered for type: %s", brainType))
	}
}

// NewBrain creates a new brain instance based on the specified type.
// For GenerativeBrain types, an AIService instance must be provided via config.
func NewBrain(brainType BrainType, config ...interface{}) (Brain, error) {
	switch brainType {
	case BrainTypeHitchhiker, BrainTypeFixed, BrainTypeClaudeCLI:
		factoryInterface, ok := simpleBrainFactories.Load(brainType)
		if !ok {
			return nil, fmt.Errorf("unsupported or unregistered hitchhiker/fixed brain type: %s", brainType)
		}
		factory, ok := factoryInterface.(SimpleBrainFactoryFunc)
		if !ok {
			return nil, fmt.Errorf("invalid factory function registered for hitchhiker/fixed brain type: %s", brainType)
		}
		return factory()

	case BrainTypeGenerative, BrainTypeLlamaCPP:
		if len(config) == 0 || config[0] == nil {
			return nil, fmt.Errorf("an AIService instance must be provided in config[0] for %s brain", brainType)
		}
		aiService, ok := config[0].(pkgaiservice.AIService)
		if !ok {
			return nil, fmt.Errorf("config[0] must be a pkgaiservice.AIService for %s brain, got %T", brainType, config[0])
		}

		factoryInterface, ok := smartBrainFactories.Load(brainType)
		if !ok {
			return nil, fmt.Errorf("unsupported or unregistered smart brain type: %s", brainType)
		}
		factory, ok := factoryInterface.(SmartBrainFactoryFunc)
		if !ok {
			return nil, fmt.Errorf("invalid factory function registered for smart brain type: %s", brainType)
		}
		return factory(aiService)

	default:
		return nil, fmt.Errorf("unknown brain type specified: %s", brainType)
	}
}

// --- Convenience Factory for SmartBrain ---

// LLMServiceType mirrors the type defined in aifactory for convenience.
type LLMServiceType string

// Constants for supported LLM service types.
const (
	LLMServiceTypeClaude LLMServiceType = "claude"
	LLMServiceTypeOpenAI LLMServiceType = "openai"
	LLMServiceTypeGemini LLMServiceType = "gemini"
)

// NewGenerativeBrainWithService creates a new GenerativeBrain using the specified LLM service type.
// It handles the creation of the underlying AI service and calls NewBrain.
func NewGenerativeBrainWithService(serviceType LLMServiceType, apiKey, model string) (Brain, error) {
	// Use the factory package to create the service
	aiService, err := aifactory.NewService(aifactory.ServiceType(serviceType), apiKey, model)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI service for GenerativeBrain: %w", err)
	}

	// Call the generic NewBrain factory, passing the created AIService (interface) in the config.
	return NewBrain(BrainTypeGenerative, aiService)
}

// --- Error Constants ---

var (
	ErrNotInitialized     = errors.New("brain not initialized")
	ErrAlreadyInitialized = errors.New("brain already initialized")
	ErrKnowledgeNotFound  = errors.New("knowledge not found")
	ErrKnowledgeExists    = errors.New("knowledge already exists")
)
