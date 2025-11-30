package mcptools

var embeddingDimension int
var embeddingModelId string

func SetEmbeddingDimension(dim int) {
	embeddingDimension = dim
}

func GetEmbeddingDimension() int {
	return embeddingDimension
}

func SetEmbeddingModelId(modelId string) {
	embeddingModelId = modelId
}

func GetEmbeddingModelId() string {
	return embeddingModelId
}
