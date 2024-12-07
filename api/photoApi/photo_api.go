package photoapi

import (
	"github.com/culbec/go-rest-api/db"
	"github.com/culbec/go-rest-api/types"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PhotoHandler struct {
	db *db.Client
}

func NewPhotoHandler(db *db.Client) *PhotoHandler {
	return &PhotoHandler{
		db: db,
	}
}

func (h *PhotoHandler) GetPhotosOfUser(ctx *gin.Context) {
	user_id := ctx.Param("user_id")

	if user_id == "" {
		ctx.JSON(400, gin.H{"message": "User ID not provided"})
		return
	}

	var photos = make([]types.Photo, 0)

	filter := &bson.D{{Key: "user_id", Value: user_id}}

	status, err := h.db.QueryCollection("photos", filter, nil, &photos)
	if err != nil {
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	ctx.IndentedJSON(status, photos)
}

func (h *PhotoHandler) AddPhoto(ctx *gin.Context) {
	var photo types.Photo

	if err := ctx.ShouldBindJSON(&photo); err != nil {
		ctx.JSON(400, gin.H{"message": err.Error()})
		return
	}

	id, status, err := h.db.InsertDocument("photos", nil, &photo)
	if err != nil {
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	if id == "" {
		ctx.JSON(500, gin.H{"message": "Error inserting the document"})
		return
	}

	photo.ID = id.(primitive.ObjectID)

	ctx.IndentedJSON(status, photo)
}

func (h *PhotoHandler) DeletePhoto(ctx *gin.Context) {
	filepath := ctx.Param("filepath")

	if filepath == "" {
		ctx.JSON(400, gin.H{"message": "Filepath not provided"})
		return
	}

	filter := &bson.D{{Key: "filepath", Value: filepath}}

	status, err := h.db.DeleteDocument("photos", filter)
	if err != nil {
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	ctx.JSON(status, gin.H{"message": "Photo deleted"})
}
