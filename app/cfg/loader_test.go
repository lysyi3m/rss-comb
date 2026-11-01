package cfg

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Cfg
		wantErr bool
	}{
		{"valid", Cfg{WorkerCount: 1, SchedulerInterval: 1}, false},
		{"zero workers", Cfg{WorkerCount: 0, SchedulerInterval: 30}, true},
		{"negative workers", Cfg{WorkerCount: -1, SchedulerInterval: 30}, true},
		{"zero interval", Cfg{WorkerCount: 5, SchedulerInterval: 0}, true},
		{"negative interval", Cfg{WorkerCount: 5, SchedulerInterval: -1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
