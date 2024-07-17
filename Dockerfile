FROM golang

# Copy the local package files to the container's workspace
ADD . /go/src/github.com/sergingroisman/meal-maker-functions

# bake in some environment variables?
# ENV SOME_ENV ""

# Set the working directory to avoid relative paths after this
WORKDIR /go/src/github.com/sergingroisman/meal-maker-functions

# Fetch the dependencies
RUN go get .

# build the binary to run later
RUN go build handler.go

# Run the command by default when the container starts
ENTRYPOINT /go/bin/program

# Document that the service listens on port 3000
EXPOSE 8080