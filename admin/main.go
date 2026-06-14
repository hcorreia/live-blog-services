package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"hcorreia/live-blog/admin/components"
	"hcorreia/live-blog/common"
	pb "hcorreia/live-blog/common/blog"
)

var theme = components.Theme{
	SiteName: "Test PB",
	Lang:     "en",
}

// func ThemeMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		ctx := context.WithValue(r.Context(), "theme", theme)
// 		ctx = context.WithValue(ctx, "color", "red")

// 		next.ServeHTTP(w, r.WithContext(ctx))
// 	})
// }

func ThemeMiddleware(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return (func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "theme", theme)
		ctx = context.WithValue(ctx, "color", "red")

		next(w, r.WithContext(ctx))
	})
}

func getPosts(ctx context.Context, client pb.BlogServiceClient) ([]*pb.PostResponse, error) {
	posts, err := client.ListPosts(ctx, &pb.ListParams{Page: 1})
	if err != nil {
		return nil, err
	}

	for _, item := range posts.GetPosts() {
		fmt.Println("Post: ", item.ID, item.Title)
	}

	fmt.Println("Posts: ", posts)

	return posts.GetPosts(), nil
}

func getPost(ctx context.Context, client pb.BlogServiceClient, id int32) (*pb.PostResponse, error) {
	post, err := client.GetPost(ctx, &pb.IdParam{Id: id})
	if err != nil {
		return nil, err
	}

	fmt.Println("Post: ", post)

	return post, nil
}

func newPost(
	ctx context.Context,
	client pb.BlogServiceClient,
	data any,
) (*pb.PostResponse, error) {

	fmt.Println("Data: ", data)

	d, ok := data.(struct {
		Title   string
		Image   string
		Content string
	})

	if !ok {
		return nil, errors.New("invalid data type!")
	}

	post, err := client.CreatePost(ctx, &pb.PostCreateRequest{
		Title:   d.Title,
		Image:   d.Image,
		Content: d.Content,
	})
	if err != nil {
		return nil, err
	}

	fmt.Println("Post: ", post)

	return post, nil
}

func updatePost(
	ctx context.Context,
	client pb.BlogServiceClient,
	post *pb.PostResponse,
	data any,
) (*pb.PostResponse, error) {

	fmt.Println("Data: ", data)

	d, ok := data.(struct {
		Title   string
		Image   string
		Content string
	})

	if !ok {
		return nil, errors.New("invalid data type!")
	}

	post, err := client.UpdatePost(ctx, &pb.PostUpdateRequest{
		ID:      post.ID,
		Title:   d.Title,
		Image:   d.Image,
		Content: d.Content,
	})
	if err != nil {
		return nil, err
	}

	fmt.Println("Post: ", post)

	return post, nil
}

func deletePost(
	ctx context.Context,
	client pb.BlogServiceClient,
	id int32,
) error {

	fmt.Println("ID: ", id)

	_, err := client.DeletePost(ctx, &pb.IdParam{Id: id})

	return err
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

	mux.HandleFunc("GET /{$}", ThemeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		components.Home("Test...").Render(r.Context(), w)

		fmt.Println("Homepage")
	}))
	mux.HandleFunc("GET /posts/{$}", ThemeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		result, err := getPosts(ctx, c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		components.PostList(result).Render(r.Context(), w)

		fmt.Println("Posts: %s", result)
	}))
	mux.HandleFunc("GET /posts/{id}/{$}", ThemeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, err.Error())
			return
		}

		result, err := getPost(ctx, c, int32(id))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, err.Error())
			return
		}

		components.PostDetail(result).Render(r.Context(), w)

		fmt.Println("Post: %s", result)
	}))
	mux.HandleFunc("GET /posts/new/{$}", ThemeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		components.PostForm(&pb.PostResponse{}, map[string]string{}, map[string]string{}).Render(r.Context(), w)

		fmt.Println("Post: %s", &pb.PostResponse{})
	}))
	mux.HandleFunc("POST /posts/{$}", ThemeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Validation

		data := map[string]string{
			"Title":   r.FormValue("title"),
			"Image":   r.FormValue("image"),
			"Content": r.FormValue("content"),
		}

		result, err := newPost(ctx, c, struct {
			Title   string
			Image   string
			Content string
		}{
			Title:   data["Title"],
			Image:   data["Image"],
			Content: data["Content"],
		})
		if err != nil {
			fmt.Println("Post new ERROR: %s", err)

			errors := map[string]string{
				"__FORM__": "Failed to create post: " + err.Error(),
			}

			fmt.Printf("Post new ERRORS: %v\n", errors)
			fmt.Printf("Post new ERROR__FORM__: %v\n", errors["__FORM__"])
			fmt.Printf("Post new ERROR__FORM__2: %v\n", errors["__FORM__2"] == "")

			w.WriteHeader(http.StatusBadRequest)
			components.PostForm(&pb.PostResponse{}, data, errors).Render(r.Context(), w)
			return
		}

		fmt.Println("Post new OK: %s", result)
		fmt.Println("Post: %s", result)

		http.Redirect(w, r, fmt.Sprintf("/posts/%d/", result.ID), http.StatusSeeOther)
	}))
	mux.HandleFunc("GET /posts/{id}/edit/{$}", ThemeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, err.Error())
			return
		}

		result, err := getPost(ctx, c, int32(id))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, err.Error())
			return
		}

		components.PostForm(result, map[string]string{
			"Title":   result.Title,
			"Image":   result.Image,
			"Content": result.Content,
		}, map[string]string{}).Render(r.Context(), w)

		fmt.Println("Post: %s", result)
	}))
	// Should be PUT
	mux.HandleFunc("POST /posts/{id}/{$}", ThemeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Validation

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, err.Error())
			return
		}

		post, err := getPost(ctx, c, int32(id))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, err.Error())
			return
		}

		data := map[string]string{
			"Title":   r.FormValue("title"),
			"Image":   r.FormValue("image"),
			"Content": r.FormValue("content"),
		}

		result, err := updatePost(ctx, c, post, struct {
			Title   string
			Image   string
			Content string
		}{
			Title:   data["Title"],
			Image:   data["Image"],
			Content: data["Content"],
		})
		if err != nil {
			fmt.Println("Post update ERROR: %s", err)

			errors := map[string]string{
				"__FORM__": "Failed to update post: " + err.Error(),
			}

			fmt.Printf("Post update ERRORS: %v\n", errors)
			fmt.Printf("Post update ERROR__FORM__: %v\n", errors["__FORM__"])
			fmt.Printf("Post update ERROR__FORM__2: %v\n", errors["__FORM__2"] == "")

			w.WriteHeader(http.StatusBadRequest)
			components.PostForm(&pb.PostResponse{}, data, errors).Render(r.Context(), w)
			return
		}

		fmt.Println("Post update OK: %s", result)
		fmt.Println("Post: %s", result)

		http.Redirect(w, r, fmt.Sprintf("/posts/%d/", result.ID), http.StatusSeeOther)
	}))
	// Should be DELETE /posts/{id}/
	mux.HandleFunc("POST /posts/{id}/delete/{$}", ThemeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, err.Error())
			return
		}

		if err := deletePost(ctx, c, int32(id)); err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, err.Error())
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/posts/"), http.StatusSeeOther)

	}))
	mux.HandleFunc("GET /health/{$}", func(w http.ResponseWriter, r *http.Request) {
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
