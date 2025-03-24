package fsm

type Option func(*FSM)

func WithLogger(logger Logger) Option {
	return func(f *FSM) {
		f.logger = logger
	}
}

func WithStateStorage(storage StateStorage) Option {
	return func(f *FSM) {
		f.storage = storage
	}
}

func WithTransitionHook(hook TransitionHook) Option {
	return func(f *FSM) {
		f.transitionHook = hook
	}
}

func WithAutoLock(storage LockableStorage, cfg LockRetryConfig, onFail LockFailureHandler) Option {
	return func(f *FSM) {
		f.lockableStorage = storage
		f.lockRetry = cfg
		f.lockFailureHandler = onFail

		if f.storage == nil {
			f.storage = storage
		}
	}
}
