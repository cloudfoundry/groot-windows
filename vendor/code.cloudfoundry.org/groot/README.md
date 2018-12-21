# groot

A framework for building image plugins for
[Guardian](https://github.com/cloudfoundry/guardian) in Go.

## Running tests
Running Groot tests is as easy as running `scripts/test`. Some of the tests are testing against a private docker registry and therefore you need to setup your own (see below for instructions how to do that).
The following environment variables configure the private docker registry access:
* `DOCKER_REGISTRY_USERNAME` - the private registry username
* `DOCKER_REGISTRY_PASSWORD` - the private registry password
* `PRIVATE_DOCKER_IMAGE_URL` - the private docker image URL, e.g. `docker://my-user/my-image:my-tag`

## Setting up the private docker image
* Build and push the docker image from `fetcher/layerfetcher/source/assets/groot-private-docker-image/Dockerfile`:
```
docker build -t my-user/my-image:my-tag fetcher/layerfetcher/source/assets/groot-private-docker-image
docker login
docker push my-user/my-image:my-tag
```
* Make sure that the image is private (e.g. via logging into the registry admin UI)
