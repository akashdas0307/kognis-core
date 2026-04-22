package schema

// Known capability IDs that plugins can declare.
const (
	CapPerception   = "PERCEPTION"
	CapCognition    = "COGNITION"
	CapAction       = "ACTION"
	CapReflection   = "REFLECTION"
	CapDream        = "DREAM"
	CapMemory       = "MEMORY"
	CapEmotion      = "EMOTION"
	CapToolBridge   = "TOOL_BRIDGE"
	CapLanguage     = "LANGUAGE"
	CapHealth       = "HEALTH"
)

// Known pipeline names.
const (
	PipelinePerceptionCognition = "PERCEPTION_TO_COGNITION"
	PipelineCognitionAction     = "COGNITION_TO_ACTION"
	PipelineCognitionReflection = "COGNITION_TO_REFLECTION"
	PipelineReflectionCognition = "REFLECTION_TO_COGNITION"
	PipelineCognitionDream      = "COGNITION_TO_DREAM"
	PipelineDreamCognition      = "DREAM_TO_COGNITION"
	PipelineCognitionLanguage   = "COGNITION_TO_LANGUAGE"
)

// Pipeline slot names within each pipeline.
const (
	SlotPerceive     = "perceive"
	SlotPreprocess   = "preprocess"
	SlotEnrich       = "enrich"
	SlotThink        = "think"
	SlotDecide       = "decide"
	SlotAct          = "act"
	SlotExecute      = "execute"
	SlotReflect      = "reflect"
	SlotConsolidate  = "consolidate"
	SlotDreamReplay  = "dream_replay"
	SlotDreamConsolidate = "dream_consolidate"
	SlotGenerate     = "generate"
	SlotRender       = "render"
)

// Plugin lifecycle states matching registry.PluginState.
const (
	StateUnregistered  = "UNREGISTERED"
	StateRegistered    = "REGISTERED"
	StateStarting      = "STARTING"
	StateHealthyActive = "HEALTHY_ACTIVE"
	StateUnhealthy     = "UNHEALTHY"
	StateUnresponsive  = "UNRESPONSIVE"
	StateCircuitOpen   = "CIRCUIT_OPEN"
	StateDead          = "DEAD"
	StateShuttingDown  = "SHUTTING_DOWN"
	StateShutDown      = "SHUT_DOWN"
)