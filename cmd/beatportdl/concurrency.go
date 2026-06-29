package main

import (
	"fmt"
	"sync"
)

func (app *application) globalWorker(fn func()) {
	app.wg.Add(1)

	go func() {
		app.semAcquire(app.globalSem)
		defer app.wg.Done()
		defer app.semRelease(app.globalSem)
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf(fmt.Errorf("%s", err).Error())
			}
		}()
		fn()
	}()
}

func (app *application) downloadWorker(wg *sync.WaitGroup, fn func()) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		select {
		case <-app.ctx.Done():
			return
		default:
		}

		app.semAcquire(app.downloadSem)
		defer app.semRelease(app.downloadSem)

		defer func() {
			if err := recover(); err != nil {
				fmt.Printf(fmt.Errorf("%s", err).Error())
			}
		}()
		fn()
	}()
}

func (app *application) semAcquire(s chan struct{}) {
	s <- struct{}{}
}

func (app *application) semRelease(s chan struct{}) {
	<-s
}
