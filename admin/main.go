package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"hcorreia/live-blog/common"
	pb "hcorreia/live-blog/common/blog"
)

func getPosts(ctx context.Context, client pb.BlogServiceClient) (*pb.PostListResponse, error) {
	posts, err := client.ListPosts(ctx, &pb.Params{Page: 1})
	if err != nil {
		return nil, err
	}

	fmt.Println("Posts: ", posts)

	return posts, nil
}

func main() {
	conn, err := grpc.NewClient(
		common.Env.BlogServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalln("Unable to connect to server", common.Env.BlogServiceAddr, err)
	}
	defer conn.Close()

	log.Println("Connected to", common.Env.BlogServiceAddr)

	c := pb.NewBlogServiceClient(conn)

	ctx := context.Background()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		result, err := getPosts(ctx, c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		fmt.Fprintf(w, "Posts: %s", result.Posts)
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_, err := getPosts(ctx, c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		fmt.Fprint(w, "Ok")
	})

	fmt.Println("Running on port", common.Env.AdminAddr)
	log.Fatal(http.ListenAndServe(common.Env.AdminAddr, mux))
}
