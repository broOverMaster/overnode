// Пакет lifecycle координирует запуск и остановку сервисов OverNode.
package lifecycle

import (
	"context"
	"errors"
)

// LifeCycle запускает сервис до отмены контекста или ошибки.
type LifeCycle interface {
	Run(context.Context) error
}

// Group хранит результаты запущенных сервисов и их общий сигнал остановки.
type Group struct {
	results <-chan error
	cancel  context.CancelFunc
	count   int
}

// Start запускает зарегистрированные сервисы с общим дочерним контекстом.
// Срез должен содержать только ненулевые сервисы. После Start нужно вызвать Wait.
func Start(ctx context.Context, lifeCycles []LifeCycle) *Group {
	ctx, cancel := context.WithCancel(ctx)
	results := make(chan error, len(lifeCycles))
	for _, service := range lifeCycles {
		go func() { results <- service.Run(ctx) }()
	}
	return &Group{results: results, cancel: cancel, count: len(lifeCycles)}
}

// Wait после завершения первого сервиса останавливает остальные и собирает ошибки.
// Вызывается один раз; для пустой группы возвращает nil без ожидания.
func (g *Group) Wait() error {
	defer g.cancel()
	var errs []error
	for i := 0; i < g.count; i++ {
		err := <-g.results
		g.cancel()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
