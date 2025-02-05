
## Docker Usage

### Building the Docker Image



 Navigate to the root directory of your project.
2. Run the following command to build the Docker image:

```bash

   docker build -t <image name> -f docker/Dockerfile .
   ```

  
## Running the Docker Container in the server



 ```bash
docker run -it -p 8080:8080 <name of the image>

   ```
## To Run in the container

```bash

docker container run -d -p 8080:8080 --name forum <name of container>