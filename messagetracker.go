package runsqs

import (
	"context"
	"time"
)

type MessageWorker struct {
	DB MessageTrackerBackend
}

// GetOrPutMessage placeholder
func (mw *MessageWorker) GetOrPutMessage(ctx context.Context, id string) (bool, error) {
	mm, err := mw.DB.GetMessage(ctx, id)
	if err != nil {
		return false, err
	} else if mm == nil {
		mw.DB.PutNewMessage(ctx, &SQSMessage{
			ID:        id,
			Status:    Processing,
			UpdatedAt: time.Now(),
		})
	} else if mm.Status == Processing || mm.Status == WaitingToRetry {
		return false, nil
	}

	return true, nil
}

func (mw *MessageWorker) UpdateMessageStatus(ctx context.Context, status MessageStatus) error {
	return mw.DB.UpdateMessageStatus(ctx, status)
}
