package vectorstores

import (
	"context"
	"fmt"
	"strings"

	"github.com/pinecone-io/go-pinecone/pinecone"
	"github.com/tmc/langchaingo/retrieval/embeddings"
	"github.com/tmc/langchaingo/retrieval/loaders"
	"google.golang.org/protobuf/types/known/structpb"
)

// PineconeVectorStore implements VectorStore interface using Pinecone
type PineconeVectorStore struct {
	client     *pinecone.Client
	index      *pinecone.IndexConnection
	indexName  string
	namespace  string
	embeddings embeddings.Embeddings
	dimension  int
}

// PineconeConfig holds configuration for Pinecone vector store
type PineconeConfig struct {
	// APIKey is the Pinecone API key
	APIKey string

	// Environment is the Pinecone environment (e.g., "us-west1-gcp")
	Environment string

	// IndexName is the name of the Pinecone index
	IndexName string

	// Namespace is the namespace within the index (optional)
	Namespace string

	// Dimension is the vector dimension
	Dimension int

	// Metric is the distance metric: "cosine", "euclidean", "dotproduct"
	Metric string

	// AutoCreateIndex creates index if it doesn't exist
	AutoCreateIndex bool
}

// NewPineconeVectorStore creates a new PineconeVectorStore
func NewPineconeVectorStore(
	config PineconeConfig,
	emb embeddings.Embeddings,
) (*PineconeVectorStore, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if config.IndexName == "" {
		return nil, fmt.Errorf("index name is required")
	}

	if config.Dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}

	if config.Metric == "" {
		config.Metric = "cosine"
	}

	// Create Pinecone client
	ctx := context.Background()
	client, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: config.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Pinecone client: %w", err)
	}

	store := &PineconeVectorStore{
		client:     client,
		indexName:  config.IndexName,
		namespace:  config.Namespace,
		embeddings: emb,
		dimension:  config.Dimension,
	}

	// Create or get index
	if config.AutoCreateIndex {
		err = store.createOrGetIndex(ctx, config.Metric)
	} else {
		err = store.getIndex(ctx)
	}

	if err != nil {
		return nil, err
	}

	return store, nil
}

// createOrGetIndex creates index if not exists
func (store *PineconeVectorStore) createOrGetIndex(ctx context.Context, metric string) error {
	// Check if index exists
	_, err := store.client.DescribeIndex(ctx, store.indexName)
	if err == nil {
		// Index exists, get connection
		return store.getIndex(ctx)
	}

	// Convert metric to Pinecone format
	var pineconeMetric pinecone.IndexMetric
	switch strings.ToLower(metric) {
	case "cosine":
		pineconeMetric = pinecone.Cosine
	case "euclidean":
		pineconeMetric = pinecone.Euclidean
	case "dotproduct":
		pineconeMetric = pinecone.Dotproduct
	default:
		pineconeMetric = pinecone.Cosine
	}

	// Create index
	_, err = store.client.CreateServerlessIndex(ctx, &pinecone.CreateServerlessIndexRequest{
		Name:      store.indexName,
		Dimension: int32(store.dimension),
		Metric:    pineconeMetric,
		Cloud:     pinecone.Aws,
		Region:    "us-east-1",
	})
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Wait for index to be ready
	_, err = store.client.DescribeIndex(ctx, store.indexName)
	if err != nil {
		return fmt.Errorf("failed to verify index creation: %w", err)
	}

	return store.getIndex(ctx)
}

// getIndex gets connection to existing index
func (store *PineconeVectorStore) getIndex(ctx context.Context) error {
	idx, err := store.client.DescribeIndex(ctx, store.indexName)
	if err != nil {
		return fmt.Errorf("failed to describe index: %w", err)
	}

	// Get index connection
	idxConnection, err := store.client.Index(pinecone.NewIndexConnParams{
		Host: idx.Host,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to index: %w", err)
	}

	store.index = idxConnection
	return nil
}

// AddDocuments adds documents to the vector store
func (store *PineconeVectorStore) AddDocuments(
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

	// Prepare vectors for Pinecone
	ids := make([]string, len(docs))
	pineconeVectors := make([]*pinecone.Vector, len(docs))

	for i, doc := range docs {
		// Generate unique ID
		ids[i] = generateID()

		// Convert float64 to float32
		values := make([]float32, len(vectors[i]))
		for j, val := range vectors[i] {
			values[j] = float32(val)
		}

		// Prepare metadata
		metadata := make(map[string]interface{})
		metadata["text"] = doc.Content // Store text in metadata
		if doc.Metadata != nil {
			for k, v := range doc.Metadata {
				metadata[k] = v
			}
		}

		// Convert metadata to structpb
		metadataStruct, err := structpb.NewStruct(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to convert metadata: %w", err)
		}

		pineconeVectors[i] = &pinecone.Vector{
			Id:       ids[i],
			Values:   values,
			Metadata: metadataStruct,
		}
	}

	// Upsert vectors
	_, err = store.index.UpsertVectors(ctx, pineconeVectors)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert vectors: %w", err)
	}

	return ids, nil
}

// SimilaritySearch performs similarity search
func (store *PineconeVectorStore) SimilaritySearch(
	ctx context.Context,
	query string,
	k int,
) ([]*loaders.Document, error) {
	return store.SimilaritySearchWithScore(ctx, query, k, 0.0)
}

// SimilaritySearchWithScore performs similarity search with score threshold
func (store *PineconeVectorStore) SimilaritySearchWithScore(
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

	// Query Pinecone
	response, err := store.index.QueryByVectorValues(ctx, &pinecone.QueryByVectorValuesRequest{
		Vector:          queryVector32,
		TopK:            uint32(k),
		Namespace:       store.namespace,
		IncludeMetadata: true,
		IncludeValues:   false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query Pinecone: %w", err)
	}

	// Parse results
	docs := make([]*loaders.Document, 0)

	for _, match := range response.Matches {
		// Apply score threshold
		score := float64(match.Score)
		if scoreThreshold > 0 && score < scoreThreshold {
			continue
		}

		// Extract metadata
		metadata := make(map[string]any)
		var text string

		if match.Vector.Metadata != nil {
			for k, v := range match.Vector.Metadata.AsMap() {
				if k == "text" {
					if str, ok := v.(string); ok {
						text = str
					}
				} else {
					metadata[k] = v
				}
			}
		}

		// Add score and ID
		metadata["score"] = score
		metadata["id"] = match.Vector.Id

		doc := loaders.NewDocument(text, metadata)
		docs = append(docs, doc)
	}

	return docs, nil
}

// Delete removes documents by IDs
func (store *PineconeVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	err := store.index.DeleteVectorsById(ctx, ids)
	if err != nil {
		return fmt.Errorf("failed to delete vectors: %w", err)
	}

	return nil
}

// Count returns the number of documents in the index
func (store *PineconeVectorStore) Count(ctx context.Context) (int64, error) {
	stats, err := store.index.DescribeIndexStats(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get index stats: %w", err)
	}

	if store.namespace != "" {
		// Get count for specific namespace
		if ns, ok := stats.Namespaces[store.namespace]; ok {
			return int64(ns.VectorCount), nil
		}
		return 0, nil
	}

	// Get total count across all namespaces
	return int64(stats.TotalVectorCount), nil
}

// GetByIDs retrieves documents by their IDs
func (store *PineconeVectorStore) GetByIDs(
	ctx context.Context,
	ids []string,
) ([]*loaders.Document, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Fetch vectors by IDs
	response, err := store.index.FetchVectors(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vectors: %w", err)
	}

	docs := make([]*loaders.Document, 0, len(ids))

	for _, id := range ids {
		vector, ok := response.Vectors[id]
		if !ok {
			continue
		}

		// Extract metadata
		metadata := make(map[string]any)
		var text string

		if vector.Metadata != nil {
			for k, v := range vector.Metadata.AsMap() {
				if k == "text" {
					if str, ok := v.(string); ok {
						text = str
					}
				} else {
					metadata[k] = v
				}
			}
		}

		metadata["id"] = id

		doc := loaders.NewDocument(text, metadata)
		docs = append(docs, doc)
	}

	return docs, nil
}

// Clear removes all documents from the namespace
func (store *PineconeVectorStore) Clear(ctx context.Context) error {
	// Delete all vectors in namespace
	err := store.index.DeleteAllVectorsInNamespace(ctx, store.namespace)
	if err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}

	return nil
}

// Close closes the connection to Pinecone
func (store *PineconeVectorStore) Close() error {
	// Pinecone Go client doesn't require explicit close
	return nil
}
