package internal

const (
	// Logging settings
	LoggingFormat    = "logging.format"
	LoggingLevel     = "logging.level"
	LoggingEnableRpc = "logging.rpc"

	// Backoff settings
	BackoffInitInterval   = "backoff.initialInterval"
	BackoffMaxInterval    = "backoff.maxInterval"
	BackoffMultiplier     = "backoff.multiplier"
	BackoffRandFactor     = "backoff.randFactor"
	BackoffMaxElapsedTime = "backoff.maxElapsedTime"

	// Service settings
	SwaggerUIPrefix    = "api.swaggerui"
	BackendPrefix      = "api.backend"
	FrontendPrefix     = "api.frontend"
	MmlogicPrefix      = "api.mmlogic"
	SynchronizerPrefix = "api.synchronizer"
	EvaluatorPrefix    = "api.evaluator"

	// TODO: Move these configs to api.synchronizer section
	SynchronizerEnabled              = "synchronizer.enabled"
	SynchronizerRegistrationMs       = "synchronizer.registrationIntervalMs"
	SynchronizerProposalCollectionMs = "synchronizer.proposalCollectionIntervalMs"

	GRPCPortSuffix = ".grpcport"
	HostNameSuffix = ".hostname"
	HTTPPortSuffix = ".httpport"

	// Statestore settings
	StatestorePageSize = "storage.page.size"

	// Redis settings
	RedisConnMaxIdle            = "redis.pool.maxIdle"
	RedisConnMaxActive          = "redis.pool.maxActive"
	RedisConnIdleTimeout        = "redis.pool.idleTimeout"
	RedisConnHealthCheckTimeout = "redis.pool.healthCheckTimeout"
	RedisIgnoreListTimeToLive   = "redis.ignoreLists.ttl"
	RedisExpiration             = "redis.expiration"
	RedisUser                   = "redis.user"
	RedisPassword               = "redis.password"
	RedisHostName               = "redis.hostname"
	RedisPort                   = "redis.port"

	// Ticket Indices
	TicketIndices = "ticketIndices"
)
