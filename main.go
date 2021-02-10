package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	h "github.com/hex-microservice/api"
	mr "github.com/hex-microservice/repository/mongo"
	rr "github.com/hex-microservice/repository/redis"
	"github.com/hex-microservice/shortener"
)

func chooseRepository() shortener.RedirectRepository {
	switch os.Getenv("URL_DB") {
	case "redis":
		redisUrl := os.Getenv("REDIS_URL")
		repo, err := rr.NewRedisRepository(redisUrl)
		if err != nil {
			log.Fatal(err)
		}

		return repo
	case "mongo":
		mongoUrl := os.Getenv("MONGO_URL")
		mongoDb := os.Getenv("MONGO_DB")
		mongoTimeout, _ := strconv.Atoi(os.Getenv("MONGO_TIMEOUT"))
		repo, err := mr.NewMongoRepository(mongoUrl, mongoDb, mongoTimeout)

		if err != nil {
			log.Fatal(err)
		}

		return repo
	}

	return nil
}

func httpPort() string {
	port := "8000"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	return fmt.Sprintf(":%s", port)
}

func main() {
	repo := chooseRepository()
	service := shortener.NewRedirectService(repo)
	handler := h.NewHandler(service)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/{code}", handler.Get)
	r.Post("/", handler.Post)

	errs := make(chan error, 2)
	go func() {
		fmt.Println("Listening on port: 8000")
		errs <- http.ListenAndServe(httpPort(), r)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	fmt.Printf("Terminated %s", <-errs)
}
