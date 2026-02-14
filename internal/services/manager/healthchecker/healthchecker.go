package healthchecker

type HealthChecker interface {
	Check() error
	NotifyFailure()
}
