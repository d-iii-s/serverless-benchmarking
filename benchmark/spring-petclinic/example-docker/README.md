Native and JVM images can be generated using example dockerfile at `example-docker/` folder 

### *For native image*
```bash
docker build -f ./example-docker/native.Dockerfile -t petclinic-native .
```

### *For JVM*
```bash
docker build -f ./example-docker/jvm.Dockerfile -t petclinic-jvm .
```