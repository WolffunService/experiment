/****************************************************************************
 * Copyright 2019, Optimizely, Inc. and contributors                        *
 *                                                                          *
 * Licensed under the Apache License, Version 2.0 (the "License");          *
 * you may not use this file except in compliance with the License.         *
 * You may obtain a copy of the License at                                  *
 *                                                                          *
 *    http://www.apache.org/licenses/LICENSE-2.0                            *
 *                                                                          *
 * Unless required by applicable law or agreed to in writing, software      *
 * distributed under the License is distributed on an "AS IS" BASIS,        *
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. *
 * See the License for the specific language governing permissions and      *
 * limitations under the License.                                           *
 ***************************************************************************/

// Package event //
package event

import (
	"errors"
	"testing"
	"time"

	"github.com/optimizely/go-sdk/pkg/utils"

	"github.com/stretchr/testify/assert"
)

type MockDispatcher struct {
	ShouldFail bool
	Events     Queue
	Visitors   Queue
	Counter    int16
}

func (m *MockDispatcher) DispatchEvent(event LogEvent) (bool, error) {
	if m.ShouldFail {
		return false, errors.New("Failed to dispatch")
	}

	m.Events.Add(event)

	for visitor := range event.Event.Visitors {
		visitor := visitor
		m.Visitors.Add(visitor)
	}

	return true, nil
}

func NewMockDispatcher(queueSize int, shouldFail bool) *MockDispatcher {
	return &MockDispatcher{
		Events: NewInMemoryQueue(queueSize),
		Visitors: NewInMemoryQueue(queueSize),
		ShouldFail: shouldFail,
	}
}

func TestDefaultEventProcessor_ProcessImpression(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	processor := NewBatchEventProcessor()
	processor.EventDispatcher = nil
	processor.Start(exeCtx)
	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()

	processor.ProcessEvent(impression)

	assert.Equal(t, 1, processor.EventsCount())

	exeCtx.TerminateAndWait()

	assert.NotNil(t, processor.Ticker)

	assert.Equal(t, 0, processor.EventsCount())
}

func TestCustomEventProcessor_Create(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	processor := NewBatchEventProcessor(
		WithEventDispatcher(NewMockDispatcher(100, false)),
		WithQueueSize(10),
		WithFlushInterval(100),
	)
	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()

	processor.ProcessEvent(impression)

	assert.Equal(t, 1, processor.EventsCount())

	exeCtx.TerminateAndWait()

	assert.NotNil(t, processor.Ticker)

	assert.Equal(t, 0, processor.EventsCount())
}

func TestDefaultEventProcessor_LogEventNotification(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	processor := NewBatchEventProcessor(
		WithFlushInterval(100),
		WithQueueSize(100),
		WithQueue(NewInMemoryQueue(100)),
		WithEventDispatcher(NewMockDispatcher(100, false)),
		WithSDKKey("fakeSDKKey"),
	)

	var logEvent LogEvent

	id, _ := processor.OnEventDispatch(func(eventNotification LogEvent) {
		logEvent = eventNotification
	})
	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()
	conversion := BuildTestConversionEvent()

	processor.ProcessEvent(impression)
	processor.ProcessEvent(impression)
	processor.ProcessEvent(conversion)
	processor.ProcessEvent(conversion)

	assert.Equal(t, 4, processor.EventsCount())

	exeCtx.TerminateAndWait()

	assert.NotNil(t, logEvent)
	assert.Equal(t, 4, len(logEvent.Event.Visitors))

	err := processor.RemoveOnEventDispatch(id)

	assert.Nil(t, err)
}

func TestDefaultEventProcessor_ProcessBatch(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	mockDispatcher := NewMockDispatcher(100, false)
	processor := NewBatchEventProcessor(
		WithFlushInterval(100),
		WithQueueSize(100),
		WithQueue(NewInMemoryQueue(100)),
		WithEventDispatcher(mockDispatcher),
	)
	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()
	conversion := BuildTestConversionEvent()

	processor.ProcessEvent(impression)
	processor.ProcessEvent(impression)
	processor.ProcessEvent(conversion)
	processor.ProcessEvent(conversion)

	assert.Equal(t, 4, processor.EventsCount())

	exeCtx.TerminateAndWait()

	assert.NotNil(t, processor.Ticker)

	assert.Equal(t, 0, processor.EventsCount())
	assert.Equal(t, 1, mockDispatcher.Events.Size())
	assert.Equal(t, 4, mockDispatcher.Visitors.Size())
}

func TestDefaultEventProcessor_QSizeMet(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	mockDispatcher := NewMockDispatcher(100, false)
	processor := NewBatchEventProcessor(
		WithBatchSize(2),
		WithFlushInterval(1000),
		WithQueue(NewInMemoryQueue(2)),
		WithEventDispatcher(mockDispatcher),
	)
	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()
	processor.ProcessEvent(impression)
	processor.ProcessEvent(impression)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, processor.EventsCount())
	assert.Equal(t, 1, mockDispatcher.Events.Size())
	assert.Equal(t, 2, mockDispatcher.Visitors.Size())

	conversion := BuildTestConversionEvent()
	processor.ProcessEvent(conversion)
	processor.ProcessEvent(conversion)

	exeCtx.TerminateAndWait()

	assert.NotNil(t, processor.Ticker)
	assert.Equal(t, 0, processor.EventsCount())
	assert.Equal(t, 2, mockDispatcher.Events.Size())
	assert.Equal(t, 4, mockDispatcher.Visitors.Size())
}

func TestDefaultEventProcessor_FailedDispatch(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	mockDispatcher := NewMockDispatcher(100, true)
	processor := NewBatchEventProcessor(
		WithQueueSize(100),
		WithFlushInterval(100),
		WithQueue(NewInMemoryQueue(100)),
		WithEventDispatcher(mockDispatcher),
	)
	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()
	conversion := BuildTestConversionEvent()

	processor.ProcessEvent(impression)
	processor.ProcessEvent(impression)
	processor.ProcessEvent(conversion)
	processor.ProcessEvent(conversion)

	assert.Equal(t, 4, processor.EventsCount())

	exeCtx.TerminateAndWait()

	assert.NotNil(t, processor.Ticker)

	assert.Equal(t, 4, processor.EventsCount())
	assert.Equal(t, 0, mockDispatcher.Events.Size())
}

func TestBatchEventProcessor_FlushesOnClose(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	processor := NewBatchEventProcessor(
		WithQueueSize(100),
		WithQueue(NewInMemoryQueue(100)),
		WithEventDispatcher(NewMockDispatcher(100, false)),
	)
	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()
	conversion := BuildTestConversionEvent()

	processor.ProcessEvent(impression)
	processor.ProcessEvent(impression)
	processor.ProcessEvent(conversion)
	processor.ProcessEvent(conversion)

	assert.Equal(t, 4, processor.EventsCount())

	// Triggers the flush in the processor
	exeCtx.TerminateAndWait()

	assert.Equal(t, 0, processor.EventsCount())
}

func TestDefaultEventProcessor_ProcessBatchRevisionMismatch(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	mockDispatcher := NewMockDispatcher(100, false)
	processor := NewBatchEventProcessor(
		WithQueueSize(100),
		WithFlushInterval(100),
		WithQueue(NewInMemoryQueue(100)),
		WithEventDispatcher(mockDispatcher),
	)

	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()
	conversion := BuildTestConversionEvent()

	processor.ProcessEvent(impression)
	impression.EventContext.Revision = "12112121"
	processor.ProcessEvent(impression)
	processor.ProcessEvent(conversion)
	processor.ProcessEvent(conversion)

	assert.Equal(t, 4, processor.EventsCount())

	exeCtx.TerminateAndWait()

	assert.NotNil(t, processor.Ticker)

	assert.Equal(t, 0, processor.EventsCount())
	assert.Equal(t, 3, mockDispatcher.Events.Size())
	assert.Equal(t, 4, mockDispatcher.Visitors.Size())
}

func TestDefaultEventProcessor_ProcessBatchProjectMismatch(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	mockDispatcher := NewMockDispatcher(100, false)
	processor := NewBatchEventProcessor(
		WithQueueSize(100),
		WithFlushInterval(100),
		WithQueue(NewInMemoryQueue(100)),
		WithEventDispatcher(mockDispatcher),
	)

	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()
	conversion := BuildTestConversionEvent()

	processor.ProcessEvent(impression)
	impression.EventContext.ProjectID = "121121211111"
	processor.ProcessEvent(impression)
	processor.ProcessEvent(conversion)
	processor.ProcessEvent(conversion)

	assert.Equal(t, 4, processor.EventsCount())

	exeCtx.TerminateAndWait()

	assert.NotNil(t, processor.Ticker)

	assert.Equal(t, 0, processor.EventsCount())
	assert.Equal(t, 3, mockDispatcher.Events.Size())
	assert.Equal(t, 4, mockDispatcher.Visitors.Size())
}

func TestChanQueueEventProcessor_ProcessImpression(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	mockDispatcher := NewMockDispatcher(100, false)
	processor := NewBatchEventProcessor(
		WithQueueSize(100),
		WithFlushInterval(100),
		WithQueue(NewInMemoryQueue(100)),
		WithEventDispatcher(mockDispatcher),
	)

	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()

	processor.ProcessEvent(impression)
	processor.ProcessEvent(impression)
	processor.ProcessEvent(impression)

	exeCtx.TerminateAndWait()
	assert.NotNil(t, processor.Ticker)

	assert.Equal(t, 0, processor.EventsCount())
}

func TestChanQueueEventProcessor_ProcessBatch(t *testing.T) {
	exeCtx := utils.NewCancelableExecutionCtx()
	mockDispatcher := NewMockDispatcher(100, false)
	processor := NewBatchEventProcessor(
		WithQueueSize(100),
		WithFlushInterval(100),
		WithQueue(NewInMemoryQueue(100)),
		WithEventDispatcher(mockDispatcher),
	)
	processor.Start(exeCtx)

	impression := BuildTestImpressionEvent()
	conversion := BuildTestConversionEvent()

	processor.ProcessEvent(impression)
	processor.ProcessEvent(impression)
	processor.ProcessEvent(conversion)
	processor.ProcessEvent(conversion)

	exeCtx.TerminateAndWait()

	assert.NotNil(t, processor.Ticker)

	assert.Equal(t, 0, processor.EventsCount())
	assert.Equal(t, 1, mockDispatcher.Events.Size())
	assert.Equal(t, 4, mockDispatcher.Visitors.Size())
}
