## Docker Usage

### Building the Docker Image

Navigate to the root directory of your project. 2. Run the following command to build the Docker image:

```bash

   docker build -t name -f docker/Dockerfile .
```

## Running the Docker Container

```bash
docker run -it -p 8080:8080 name

```
