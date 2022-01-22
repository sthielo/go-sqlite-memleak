package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sthielo/go-sqlite-memleak/pkg/internal/database"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	database.InitDB()
	defer database.MyDb.Close()
	database.FillInDummyData()

	// setup context and signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	// web srv setup
	manageRouter := gin.New()
	manageRouter.Use(
		gin.Recovery(),
	)
	manageRouter.GET("/dumpdb", dumpDb)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		_, _ = os.Stdout.WriteString(">>> oom: " + ("start listening on port 8890 ...") + "\n")
		srv := &http.Server{Addr: ":8890", Handler: manageRouter}
		g.Go(func() error {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		})

		<-gCtx.Done()
		_, _ = os.Stdout.WriteString(">>> oom: " + ("stop listening ...") + "\n")
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		srv.SetKeepAlivesEnabled(false) //close idle connections
		return srv.Shutdown(timeoutCtx)
	})

	// Listen for the interrupt signal.
	<-ctx.Done()
	stop()

	if err := g.Wait(); err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("error on shutdown: %+v", err) + "\n")
		os.Exit(1)
	}
}

/**
 * handler for the only web end point ...
 */
func dumpDb(c *gin.Context) {
	err := database.Export()
	if err != nil {
		_, _ = os.Stdout.WriteString(">>> oom: " + fmt.Sprintf("error when dumping db: %+v\n", err) + "\n")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}
