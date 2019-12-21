# resize-me
I'm using "github.com/nfnt/resize" to resize the image
The server listens on port 8080
Try it out http://localhost:8080/thumbnail?url=https://www.abc.net.au/news/image/7672290-3x2-940x627.jpg&width=940&height=597

The server saves locally all images once, at all types of sizes requested by the user.

The server accepts Get /thumbnail and expects url,width and height query params.
The url is expected to be full path to image file of type jpg.
The width and height are expected to be untyped int.

Image requested for the first time at specific size is saved on the server locally.
The file is named as it's original width and height + name exp. image aaa.jpg of size 320X450 is saved as w_320h_450aaa.jpg.
after saving the original image with its dimentions, the server resizes the image to the requested width and height using github.com/nfnt/resize Thumbnail func.

The server has a map to keep track of all images it currently holds to reduce the resizing of already resized images.
