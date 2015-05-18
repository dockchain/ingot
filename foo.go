package main
import (
	"github.com/fsouza/go-dockerclient"
	"time"
	"fmt")

// https://github.com/dgrijalva/jwt-go

func main() {
	const a string = "initial"
	fmt.Println(a)

	const b, c int = 1, 2
	fmt.Println(b, c)

	const d = true
	fmt.Println(d)

	const e int = 0
	fmt.Println(e)

	f := "foo"
	fmt.Println(f)

	f = "bar"
	fmt.Println(f)



	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)

	fmt.Println("Made ", client, " err ", err)

	listener := make(chan *docker.APIEvents, 10)

	go func() {
		for evt := range listener {
			fmt.Println(evt)
		}
	}()


	client.AddEventListener(listener)
	// imgs, _ := client.ListImages(docker.ListImagesOptions{All: false})
	// for _, img := range imgs {
	// 	fmt.Println("ID: ", img.ID)
	// 	fmt.Println("RepoTags: ", img.RepoTags)
	// 	fmt.Println("Created: ", img.Created)
	// 	fmt.Println("Size: ", img.Size)
	// 	fmt.Println("VirtualSize: ", img.VirtualSize)
	// 	fmt.Println("ParentId: ", img.ParentID)
	// }

	time.Sleep(time.Second * 10)


	fmt.Println("Yo, dog!!")
}
