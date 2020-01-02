build:
	GO111MODULE=on GOOS=linux go build -o main *.go && zip lambda.zip main && rm main

# test:
# 	GRAPHQL_ORM_URL=https://api.muniprojects.com/graphql DEBUG=true go run 