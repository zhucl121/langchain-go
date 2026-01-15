package vectorstores

import (
	"context"
	"fmt"
	"strings"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/tmc/langchaingo/retrieval/embeddings"
	"github.com/tmc/langchaingo/retrieval/loaders"
)

// ChromaVectorStore implements VectorStore interface using Chroma
type ChromaVectorStore struct {
	client         *chroma.Client
	collection     *chroma.Collection
	collectionName string
	embeddings     embeddings.Embeddings
	metadataFields []string
}

// ChromaConfig holds configuration for Chroma vector store
type ChromaConfig struct {
	// URL is the Chroma server URL (e.g., "http://localhost:8000")
	URL string

	// CollectionName is the name of the collection
	CollectionName string

	// Metadata fields to include in documents
	MetadataFields []string

	// Distance function: "l2", "ip", "cosine"
	DistanceFunction string

	// AutoCreateCollection creates collection if it doesn't exist
	AutoCreateCollection bool
}

// NewChromaVectorStore creates a new ChromaVectorStore
func NewChromaVectorStore(
	config ChromaConfig,
	emb embeddings.Embeddings,
) (*ChromaVectorStore, error) {
	if config.URL == "" {
		config.URL = "http://localhost:8000"
	}

	if config.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	if config.DistanceFunction == "" {
		config.DistanceFunction = "l2"
	}

	// Create Chroma client
	client, err := chroma.NewClient(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Chroma client: %w", err)
	}

	store := &ChromaVectorStore{
		client:         client,
		collectionName: config.CollectionName,
		embeddings:     emb,
		metadataFields: config.MetadataFields,
	}

	// Get or create collection
	if config.AutoCreateCollection {
		err = store.createOrGetCollection(config.DistanceFunction)
	} else {
		err = store.getCollection()
	}

	if err != nil {
		return nil, err
	}

	return store, nil
}

// createOrGetCollection creates collection if not exists
func (store *ChromaVectorStore) createOrGetCollection(distanceFunc string) error {
	// Convert distance function to Chroma format
	var chromaDistance types.DistanceFunction
	switch strings.ToLower(distanceFunc) {
	case "l2":
		chromaDistance = types.L2
	case "ip":
		chromaDistance = types.IP
	case "cosine":
		chromaDistance = types.COSINE
	default:
		chromaDistance = types.L2
	}

	metadata := map[string]interface{}{
		"hnsw:space": chromaDistance,
	}

	collection, err := store.client.CreateCollection(
		store.collectionName,
		metadata,
		true, // get or create
		nil,  // default embedding function
	)
	if err != nil {
		return fmt.Errorf("failed to create/get collection: %w", err)
	}

	store.collection = collection
	return nil
}

// getCollection gets existing collection
func (store *ChromaVectorStore) getCollection() error {
	collection, err := store.client.GetCollection(store.collectionName, nil)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	store.collection = collection
	return nil
}

// AddDocuments adds documents to the vector store
func (store *ChromaVectorStore) AddDocuments(
	ctx context.Context,
	docs []*loaders.Document,
) ([]string, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	// Extract texts
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	// Generate embeddings
	vectors, err := store.embeddings.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed documents: %w", err)
	}

	// Prepare data for Chroma
	ids := make([]string, len(docs))
	metadatas := make([]map[string]interface{}, len(docs))

	for i, doc := range docs {
		// Generate unique ID
		ids[i] = generateID()

		// Prepare metadata
		metadata := make(map[string]interface{})
		if doc.Metadata != nil {
			for k, v := range doc.Metadata {
				// Chroma requires metadata values to be strings, numbers, or booleans
				switch val := v.(type) {
				case string, int, int64, float64, bool:
					metadata[k] = val
				default:
					metadata[k] = fmt.Sprintf("%v", val)
				}
			}
		}
		metadatas[i] = metadata
	}

	// Convert float64 slice to float32 for Chroma
	embeddings32 := make([][]float32, len(vectors))
	for i, vec := range vectors {
		embeddings32[i] = make([]float32, len(vec))
		for j, val := range vec {
			embeddings32[i][j] = float32(val)
		}
	}

	// Add to Chroma
	_, err = store.collection.Add(
		embeddings32,
		metadatas,
		texts,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add documents to Chroma: %w", err)
	}

	return ids, nil
}

// SimilaritySearch performs similarity search
func (store *ChromaVectorStore) SimilaritySearch(
	ctx context.Context,
	query string,
	k int,
) ([]*loaders.Document, error) {
	return store.SimilaritySearchWithScore(ctx, query, k, 0.0)
}

// SimilaritySearchWithScore performs similarity search with score threshold
func (store *ChromaVectorStore) SimilaritySearchWithScore(
	ctx context.Context,
	query string,
	k int,
	scoreThreshold float64,
) ([]*loaders.Document, error) {
	// Generate query embedding
	queryVectors, err := store.embeddings.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Convert to float32
	queryVector32 := make([]float32, len(queryVectors))
	for i, val := range queryVectors {
		queryVector32[i] = float32(val)
	}

	// Query Chroma
	results, err := store.collection.Query(
		[][]float32{queryVector32},
		int32(k),
		nil, // where
		nil, // where_document
		nil, // include (default: documents, metadatas, distances)
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Chroma: %w", err)
	}

	// Parse results
	docs := make([]*loaders.Document, 0)

	if len(results.Documents) > 0 && len(results.Documents[0]) > 0 {
		for i, docText := range results.Documents[0] {
			// Get distance
			var distance float64
			if len(results.Distances) > 0 && len(results.Distances[0]) > i {
				distance = float64(results.Distances[0][i])
			}

			// Apply score threshold (lower distance is better)
			if scoreThreshold > 0 && distance > scoreThreshold {
				continue
			}

			// Get metadata
			metadata := make(map[string]any)
			if len(results.Metadatas) > 0 && len(results.Metadatas[0]) > i {
				for k, v := range results.Metadatas[0][i] {
					metadata[k] = v
				}
			}

			// Add distance score
			metadata["distance"] = distance
			metadata["similarity_score"] = 1.0 / (1.0 + distance)

			// Get document ID
			if len(results.Ids) > 0 && len(results.Ids[0]) > i {
				metadata["id"] = results.Ids[0][i]
			}

			doc := loaders.NewDocument(docText, metadata)
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// Delete removes documents by IDs
func (store *ChromaVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	_, err := store.collection.Delete(ids, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// Count returns the number of documents in the collection
func (store *ChromaVectorStore) Count(ctx context.Context) (int64, error) {
	count, err := store.collection.Count()
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return int64(count), nil
}

// GetByIDs retrieves documents by their IDs
func (store *ChromaVectorStore) GetByIDs(
	ctx context.Context,
	ids []string,
) ([]*loaders.Document, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	results, err := store.collection.Get(
		ids,
		nil, // where
		nil, // limit
		nil, // offset
		nil, // where_document
		nil, // include
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents by IDs: %w", err)
	}

	docs := make([]*loaders.Document, 0, len(ids))

	for i, docText := range results.Documents {
		metadata := make(map[string]any)

		// Add metadata
		if len(results.Metadatas) > i {
			for k, v := range results.Metadatas[i] {
				metadata[k] = v
			}
		}

		// Add ID
		if len(results.Ids) > i {
			metadata["id"] = results.Ids[i]
		}

		doc := loaders.NewDocument(docText, metadata)
		docs = append(docs, doc)
	}

	return docs, nil
}

// Clear removes all documents from the collection
func (store *ChromaVectorStore) Clear(ctx context.Context) error {
	// Get all document IDs
	results, err := store.collection.Get(
		nil, // ids (nil = all)
		nil, // where
		nil, // limit
		nil, // offset
		nil, // where_document
		[]types.QueryEnum{types.QIDs}, // only get IDs
	)
	if err != nil {
		return fmt.Errorf("failed to get document IDs: %w", err)
	}

	// Delete all documents
	if len(results.Ids) > 0 {
		_, err = store.collection.Delete(results.Ids, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to clear collection: %w", err)
		}
	}

	return nil
}

// Close closes the connection to Chroma
func (store *ChromaVectorStore) Close() error {
	// Chroma Go client doesn't require explicit close
	return nil
}
