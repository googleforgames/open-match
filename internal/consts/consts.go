package consts

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
	SwaggerUIHTTPPort = "api.swaggerui.httpport"

	MmlogicHostName = "api.mmlogic.hostname"
	MmlogicHTTPPort = "api.mmlogic.httpport"
	MmlogicGRPCPort = "api.mmlogic.grpcport"

	SynchronizerHostName = "api.synchronizer.hostname"
	SynchronizerHTTPPort = "api.synchronizer.httpport"
	SynchronizerGRPCPort = "api.synchronizer.grpcport"

	EvaluatorHostName = "api.evaluator.hostname"
	EvaluatorHTTPPort = "api.evaluator.httpport"
	EvaluatorGRPCPort = "api.evaluator.grpcport"

	SynchronizerEnabled              = "synchronizer.enabled"
	SynchronizerRegistrationMs       = "synchronizer.registrationIntervalMs"
	SynchronizerProposalCollectionMs = "synchronizer.proposalCollectionIntervalMs"

	GRPCPortSuffix = ".grpcport"
	HostNameSuffix = ".hostname"
	HTTPPortSuffix = ".httpport"

	// Statestore settings
	StorePageSize = "storage.page.size"

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
