package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// LLMEntityExtractor 基于 LLM 的实体提取器。
type LLMEntityExtractor struct {
	chatModel chat.ChatModel
	schema    *EntitySchema
}

// NewLLMEntityExtractor 创建 LLM 实体提取器。
//
// 参数：
//   - chatModel: ChatModel 实例
//   - schema: 实体 Schema（nil 表示使用默认）
//
// 返回：
//   - *LLMEntityExtractor: 提取器实例
//
func NewLLMEntityExtractor(chatModel chat.ChatModel, schema *EntitySchema) *LLMEntityExtractor {
	return &LLMEntityExtractor{
		chatModel: chatModel,
		schema:    schema,
	}
}

// Extract 从文本中提取实体。
func (e *LLMEntityExtractor) Extract(ctx context.Context, text string) ([]Entity, error) {
	return e.ExtractWithSchema(ctx, text, e.schema)
}

// ExtractWithSchema 使用指定 Schema 提取实体。
func (e *LLMEntityExtractor) ExtractWithSchema(ctx context.Context, text string, schema *EntitySchema) ([]Entity, error) {
	prompt := e.buildEntityExtractionPrompt(text, schema)

	messages := []types.Message{
		types.NewSystemMessage("You are an expert at extracting entities from text. Always respond with valid JSON."),
		types.NewUserMessage(prompt),
	}

	result, err := e.chatModel.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM invoke failed: %w", err)
	}

	entities, err := e.parseEntityResponse(result.Content, text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return entities, nil
}

// buildEntityExtractionPrompt 构建实体提取提示词。
func (e *LLMEntityExtractor) buildEntityExtractionPrompt(text string, schema *EntitySchema) string {
	var prompt strings.Builder

	prompt.WriteString("Extract all entities from the following text.\n\n")

	if schema != nil {
		prompt.WriteString(fmt.Sprintf("Entity Type: %s\n", schema.Type))
		prompt.WriteString(fmt.Sprintf("Description: %s\n\n", schema.Description))

		if len(schema.Properties) > 0 {
			prompt.WriteString("Required Properties:\n")
			for name, prop := range schema.Properties {
				prompt.WriteString(fmt.Sprintf("- %s (%s): %s\n", name, prop.Type, prop.Description))
			}
			prompt.WriteString("\n")
		}

		if len(schema.Examples) > 0 {
			prompt.WriteString("Examples:\n")
			for _, example := range schema.Examples {
				prompt.WriteString(fmt.Sprintf("- %s (%s): %s\n", example.Name, example.Type, example.Description))
			}
			prompt.WriteString("\n")
		}
	} else {
		// 默认提示
		prompt.WriteString("Extract entities of the following types:\n")
		prompt.WriteString("- Person: People's names\n")
		prompt.WriteString("- Organization: Companies, institutions\n")
		prompt.WriteString("- Location: Places, cities, countries\n")
		prompt.WriteString("- Concept: Abstract concepts, ideas\n")
		prompt.WriteString("- Event: Events, occurrences\n")
		prompt.WriteString("- Product: Products, services\n")
		prompt.WriteString("- Technology: Technologies, tools\n\n")
	}

	prompt.WriteString("Text:\n")
	prompt.WriteString(text)
	prompt.WriteString("\n\n")

	prompt.WriteString("Output format (JSON):\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"entities\": [\n")
	prompt.WriteString("    {\n")
	prompt.WriteString("      \"type\": \"Person\",\n")
	prompt.WriteString("      \"name\": \"John Smith\",\n")
	prompt.WriteString("      \"description\": \"CEO of TechCorp\",\n")
	prompt.WriteString("      \"properties\": {\"role\": \"CEO\"},\n")
	prompt.WriteString("      \"confidence\": 0.95\n")
	prompt.WriteString("    }\n")
	prompt.WriteString("  ]\n")
	prompt.WriteString("}\n")

	return prompt.String()
}

// parseEntityResponse 解析 LLM 响应。
func (e *LLMEntityExtractor) parseEntityResponse(response string, sourceText string) ([]Entity, error) {
	// 查找 JSON 内容
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var parsed struct {
		Entities []struct {
			Type        string                 `json:"type"`
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			Properties  map[string]interface{} `json:"properties"`
			Confidence  float64                `json:"confidence"`
		} `json:"entities"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	entities := make([]Entity, 0, len(parsed.Entities))
	for i, e := range parsed.Entities {
		entity := Entity{
			ID:          fmt.Sprintf("entity-%d-%s", i, generateID(e.Name)),
			Type:        e.Type,
			Name:        e.Name,
			Description: e.Description,
			Properties:  e.Properties,
			Metadata:    make(map[string]interface{}),
			SourceText:  sourceText,
			Confidence:  e.Confidence,
		}

		if entity.Properties == nil {
			entity.Properties = make(map[string]interface{})
		}

		// 默认置信度
		if entity.Confidence == 0 {
			entity.Confidence = 0.8
		}

		entities = append(entities, entity)
	}

	return entities, nil
}

// LLMRelationExtractor 基于 LLM 的关系提取器。
type LLMRelationExtractor struct {
	chatModel chat.ChatModel
	schema    *RelationSchema
}

// NewLLMRelationExtractor 创建 LLM 关系提取器。
//
// 参数：
//   - chatModel: ChatModel 实例
//   - schema: 关系 Schema（nil 表示使用默认）
//
// 返回：
//   - *LLMRelationExtractor: 提取器实例
//
func NewLLMRelationExtractor(chatModel chat.ChatModel, schema *RelationSchema) *LLMRelationExtractor {
	return &LLMRelationExtractor{
		chatModel: chatModel,
		schema:    schema,
	}
}

// Extract 从文本中提取关系。
func (r *LLMRelationExtractor) Extract(ctx context.Context, text string, entities []Entity) ([]Relation, error) {
	return r.ExtractWithSchema(ctx, text, entities, r.schema)
}

// ExtractWithSchema 使用指定 Schema 提取关系。
func (r *LLMRelationExtractor) ExtractWithSchema(ctx context.Context, text string, entities []Entity, schema *RelationSchema) ([]Relation, error) {
	prompt := r.buildRelationExtractionPrompt(text, entities, schema)

	messages := []types.Message{
		types.NewSystemMessage("You are an expert at extracting relationships between entities. Always respond with valid JSON."),
		types.NewUserMessage(prompt),
	}

	result, err := r.chatModel.Invoke(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM invoke failed: %w", err)
	}

	relations, err := r.parseRelationResponse(result.Content, text, entities)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return relations, nil
}

// buildRelationExtractionPrompt 构建关系提取提示词。
func (r *LLMRelationExtractor) buildRelationExtractionPrompt(text string, entities []Entity, schema *RelationSchema) string {
	var prompt strings.Builder

	prompt.WriteString("Extract relationships between entities from the following text.\n\n")

	// 列出已知实体
	if len(entities) > 0 {
		prompt.WriteString("Known Entities:\n")
		for _, entity := range entities {
			prompt.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", entity.ID, entity.Name, entity.Type))
		}
		prompt.WriteString("\n")
	}

	if schema != nil {
		prompt.WriteString(fmt.Sprintf("Relation Type: %s\n", schema.Type))
		prompt.WriteString(fmt.Sprintf("Description: %s\n\n", schema.Description))

		if len(schema.SourceTypes) > 0 {
			prompt.WriteString(fmt.Sprintf("Source Types: %s\n", strings.Join(schema.SourceTypes, ", ")))
		}
		if len(schema.TargetTypes) > 0 {
			prompt.WriteString(fmt.Sprintf("Target Types: %s\n", strings.Join(schema.TargetTypes, ", ")))
		}
		prompt.WriteString("\n")

		if len(schema.Examples) > 0 {
			prompt.WriteString("Examples:\n")
			for _, example := range schema.Examples {
				prompt.WriteString(fmt.Sprintf("- %s -[%s]-> %s\n", example.Source, example.Type, example.Target))
			}
			prompt.WriteString("\n")
		}
	} else {
		// 默认提示
		prompt.WriteString("Extract relationships of the following types:\n")
		prompt.WriteString("- WORKS_FOR: Person works for Organization\n")
		prompt.WriteString("- LOCATED_IN: Entity is located in Location\n")
		prompt.WriteString("- KNOWS: Person knows Person\n")
		prompt.WriteString("- FOUNDED: Person founded Organization\n")
		prompt.WriteString("- OWNS: Person/Organization owns Entity\n")
		prompt.WriteString("- PART_OF: Entity is part of Entity\n")
		prompt.WriteString("- RELATED_TO: Entity is related to Entity\n\n")
	}

	prompt.WriteString("Text:\n")
	prompt.WriteString(text)
	prompt.WriteString("\n\n")

	prompt.WriteString("Output format (JSON):\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"relations\": [\n")
	prompt.WriteString("    {\n")
	prompt.WriteString("      \"type\": \"WORKS_FOR\",\n")
	prompt.WriteString("      \"source\": \"entity-0-john-smith\",\n")
	prompt.WriteString("      \"target\": \"entity-1-techcorp\",\n")
	prompt.WriteString("      \"description\": \"John Smith works for TechCorp as CEO\",\n")
	prompt.WriteString("      \"directed\": true,\n")
	prompt.WriteString("      \"confidence\": 0.9\n")
	prompt.WriteString("    }\n")
	prompt.WriteString("  ]\n")
	prompt.WriteString("}\n")

	return prompt.String()
}

// parseRelationResponse 解析 LLM 响应。
func (r *LLMRelationExtractor) parseRelationResponse(response string, sourceText string, entities []Entity) ([]Relation, error) {
	// 查找 JSON 内容
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var parsed struct {
		Relations []struct {
			Type        string                 `json:"type"`
			Source      string                 `json:"source"`
			Target      string                 `json:"target"`
			Description string                 `json:"description"`
			Properties  map[string]interface{} `json:"properties"`
			Directed    bool                   `json:"directed"`
			Weight      float64                `json:"weight"`
			Confidence  float64                `json:"confidence"`
		} `json:"relations"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	// 构建实体 ID 映射
	entityMap := make(map[string]bool)
	for _, entity := range entities {
		entityMap[entity.ID] = true
	}

	relations := make([]Relation, 0, len(parsed.Relations))
	for i, rel := range parsed.Relations {
		// 验证源和目标实体存在
		if !entityMap[rel.Source] || !entityMap[rel.Target] {
			continue // 跳过无效关系
		}

		relation := Relation{
			ID:          fmt.Sprintf("relation-%d-%s-%s", i, rel.Source, rel.Target),
			Type:        rel.Type,
			Description: rel.Description,
			Source:      rel.Source,
			Target:      rel.Target,
			Properties:  rel.Properties,
			Metadata:    make(map[string]interface{}),
			Weight:      rel.Weight,
			Directed:    rel.Directed,
			SourceText:  sourceText,
			Confidence:  rel.Confidence,
		}

		if relation.Properties == nil {
			relation.Properties = make(map[string]interface{})
		}

		// 默认值
		if relation.Weight == 0 {
			relation.Weight = 1.0
		}
		if relation.Confidence == 0 {
			relation.Confidence = 0.8
		}
		if !relation.Directed {
			relation.Directed = true // 默认有向
		}

		relations = append(relations, relation)
	}

	return relations, nil
}

// generateID 生成简单的 ID（用于演示，实际应使用 UUID）。
func generateID(name string) string {
	// 简化版本：将名称转换为小写并替换空格
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "'", "")
	id = strings.ReplaceAll(id, "\"", "")
	return id
}
