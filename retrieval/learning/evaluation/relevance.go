package evaluation

import (
	"time"

	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
)

// IsRelevant 默认相关性模型：判断文档是否相关
func (m *DefaultRelevanceModel) IsRelevant(docID string, queryFeedback interface{}) bool {
	qf, ok := queryFeedback.(*feedback.QueryFeedback)
	if !ok {
		return false
	}

	// 如果用户点击或阅读了文档，认为相关
	for _, fb := range qf.ImplicitFeedback {
		if fb.DocumentID == docID {
			if fb.Action == feedback.ActionClick || fb.Action == feedback.ActionRead {
				return true
			}
		}
	}

	return false
}

// GetRelevance 默认相关性模型：获取文档相关度得分
func (m *DefaultRelevanceModel) GetRelevance(docID string, queryFeedback interface{}) float64 {
	qf, ok := queryFeedback.(*feedback.QueryFeedback)
	if !ok {
		return 0
	}

	relevance := 0.0

	// 基于隐式反馈计算相关度
	for _, fb := range qf.ImplicitFeedback {
		if fb.DocumentID == docID {
			switch fb.Action {
			case feedback.ActionClick:
				relevance += 0.3
			case feedback.ActionRead:
				relevance += 0.5
				// 阅读时长加权
				if fb.Duration > 30*time.Second {
					relevance += 0.2
				}
			case feedback.ActionCopy:
				relevance += 0.4
			case feedback.ActionDownload:
				relevance += 0.6
			case feedback.ActionIgnore:
				relevance -= 0.3
			case feedback.ActionSkip:
				relevance -= 0.2
			}
		}
	}

	// 基于显式反馈调整
	for _, fb := range qf.ExplicitFeedback {
		if fb.Type == feedback.FeedbackTypePositive {
			relevance += 0.5
		} else if fb.Type == feedback.FeedbackTypeNegative {
			relevance -= 0.5
		}

		if fb.Rating > 0 {
			relevance += float64(fb.Rating) / 10.0
		}
	}

	// 归一化到 [0, 1]
	if relevance < 0 {
		relevance = 0
	}
	if relevance > 1 {
		relevance = 1
	}

	return relevance
}

// IsRelevant 基于隐式反馈的相关性模型：判断文档是否相关
func (m *ImplicitRelevanceModel) IsRelevant(docID string, queryFeedback interface{}) bool {
	return m.GetRelevance(docID, queryFeedback) > 0.5
}

// GetRelevance 基于隐式反馈的相关性模型：获取文档相关度得分
func (m *ImplicitRelevanceModel) GetRelevance(docID string, queryFeedback interface{}) float64 {
	qf, ok := queryFeedback.(*feedback.QueryFeedback)
	if !ok {
		return 0
	}

	relevance := 0.0

	for _, fb := range qf.ImplicitFeedback {
		if fb.DocumentID == docID {
			switch fb.Action {
			case feedback.ActionClick:
				relevance += m.ClickWeight
			case feedback.ActionRead:
				relevance += m.ReadWeight
				// 时长权重
				if fb.Duration > 0 {
					durationScore := float64(fb.Duration.Seconds()) / 60.0 // 归一化到分钟
					if durationScore > 1 {
						durationScore = 1
					}
					relevance += durationScore * m.DurationWeight
				}
			case feedback.ActionCopy, feedback.ActionDownload:
				relevance += 0.5
			}
		}
	}

	// 归一化
	if relevance > 1 {
		relevance = 1
	}

	return relevance
}
