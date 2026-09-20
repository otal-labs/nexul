package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCompose(t *testing.T) {
	t.Run("short syntax ports, image, env list", func(t *testing.T) {
		doc := `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
      - "127.0.0.1:9090:90"
      - "3000"
    environment:
      - FOO=bar
      - BAZ
    env_file:
      - .env.prod
`
		services, err := parseCompose([]byte(doc))
		require.NoError(t, err)
		require.Len(t, services, 1)
		svc := services[0]
		assert.Equal(t, "web", svc.Name)
		assert.Equal(t, "nginx:latest", svc.Image)
		assert.Nil(t, svc.Build)
		assert.Equal(t, []int{80, 90, 3000}, svc.Ports)
		assert.Equal(t, []string{"FOO", "BAZ", ".env.prod"}, svc.EnvKeys)
	})

	t.Run("long syntax ports, build map, env map, expose", func(t *testing.T) {
		doc := `
services:
  api:
    build:
      context: ./api
      dockerfile: Dockerfile.dev
    ports:
      - target: 8000
        published: "8000"
        protocol: tcp
    expose:
      - "9000"
    environment:
      DB_HOST: localhost
      DB_PORT: "5432"
`
		services, err := parseCompose([]byte(doc))
		require.NoError(t, err)
		require.Len(t, services, 1)
		svc := services[0]
		assert.Equal(t, "api", svc.Name)
		assert.Empty(t, svc.Image)
		require.NotNil(t, svc.Build)
		assert.Equal(t, "./api", svc.Build.Context)
		assert.Equal(t, "Dockerfile.dev", svc.Build.Dockerfile)
		assert.Equal(t, []int{8000}, svc.Ports)
		assert.Equal(t, []int{9000}, svc.Expose)
		assert.Equal(t, []string{"DB_HOST", "DB_PORT"}, svc.EnvKeys)
	})

	t.Run("build shorthand string is the context", func(t *testing.T) {
		doc := `
services:
  worker:
    build: ./worker
`
		services, err := parseCompose([]byte(doc))
		require.NoError(t, err)
		require.Len(t, services, 1)
		require.NotNil(t, services[0].Build)
		assert.Equal(t, "./worker", services[0].Build.Context)
		assert.Empty(t, services[0].Build.Dockerfile)
	})

	t.Run("multiple services keep declaration order for reachability", func(t *testing.T) {
		doc := `
services:
  frontend:
    image: nginx
  backend:
    image: acme/api
    ports:
      - "8080:8080"
  cache:
    image: redis
    expose:
      - "6379"
`
		services, err := parseCompose([]byte(doc))
		require.NoError(t, err)
		require.Len(t, services, 3)
		assert.Equal(t, []string{"frontend", "backend", "cache"}, []string{services[0].Name, services[1].Name, services[2].Name})
		reachable := firstReachable(services)
		require.NotNil(t, reachable)
		assert.Equal(t, "backend", reachable.Service)
		assert.Equal(t, 8080, reachable.Port)
	})

	t.Run("no services is not an error", func(t *testing.T) {
		services, err := parseCompose([]byte("version: \"3\"\n"))
		require.NoError(t, err)
		assert.Empty(t, services)
		assert.Nil(t, firstReachable(services))
	})

	t.Run("invalid yaml errors", func(t *testing.T) {
		_, err := parseCompose([]byte("services: [this is not a map"))
		assert.Error(t, err)
	})
}

func TestParseCompose_SkipsMisindentedService(t *testing.T) {
	doc := `
services:
    broken:
    image: docker.io/bitnami/laravel:9
    ports:
      - '8000:8000'
    ok:
        image: nginx
`
	services, err := parseCompose([]byte(doc))
	require.NoError(t, err)
	require.Len(t, services, 1)
	assert.Equal(t, "ok", services[0].Name)
}
