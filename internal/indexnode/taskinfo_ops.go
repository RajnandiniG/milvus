package indexnode

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/milvus-io/milvus-proto/go-api/v2/commonpb"
	"github.com/milvus-io/milvus/internal/proto/indexpb"
	"github.com/milvus-io/milvus/pkg/common"
	"github.com/milvus-io/milvus/pkg/log"
)

type indexTaskInfo struct {
	cancel              context.CancelFunc
	state               commonpb.IndexState
	fileKeys            []string
	serializedSize      uint64
	failReason          string
	currentIndexVersion int32
	indexStoreVersion   int64

	// task statistics
	statistic *indexpb.JobInfo
}

func (i *IndexNode) loadOrStoreIndexTask(ClusterID string, buildID UniqueID, info *indexTaskInfo) *indexTaskInfo {
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	key := taskKey{ClusterID: ClusterID, BuildID: buildID}
	oldInfo, ok := i.indexTasks[key]
	if ok {
		return oldInfo
	}
	i.indexTasks[key] = info
	return nil
}

func (i *IndexNode) loadIndexTaskState(ClusterID string, buildID UniqueID) commonpb.IndexState {
	key := taskKey{ClusterID: ClusterID, BuildID: buildID}
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	task, ok := i.indexTasks[key]
	if !ok {
		return commonpb.IndexState_IndexStateNone
	}
	return task.state
}

func (i *IndexNode) storeIndexTaskState(ClusterID string, buildID UniqueID, state commonpb.IndexState, failReason string) {
	key := taskKey{ClusterID: ClusterID, BuildID: buildID}
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	if task, ok := i.indexTasks[key]; ok {
		log.Debug("IndexNode store task state", zap.String("clusterID", ClusterID), zap.Int64("buildID", buildID),
			zap.String("state", state.String()), zap.String("fail reason", failReason))
		task.state = state
		task.failReason = failReason
	}
}

func (i *IndexNode) foreachIndexTaskInfo(fn func(ClusterID string, buildID UniqueID, info *indexTaskInfo)) {
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	for key, info := range i.indexTasks {
		fn(key.ClusterID, key.BuildID, info)
	}
}

func (i *IndexNode) storeIndexFilesAndStatistic(
	ClusterID string,
	buildID UniqueID,
	fileKeys []string,
	serializedSize uint64,
	statistic *indexpb.JobInfo,
	currentIndexVersion int32,
) {
	key := taskKey{ClusterID: ClusterID, BuildID: buildID}
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	if info, ok := i.indexTasks[key]; ok {
		info.fileKeys = common.CloneStringList(fileKeys)
		info.serializedSize = serializedSize
		info.statistic = proto.Clone(statistic).(*indexpb.JobInfo)
		info.currentIndexVersion = currentIndexVersion
		return
	}
}

func (i *IndexNode) storeIndexFilesAndStatisticV2(
	ClusterID string,
	buildID UniqueID,
	fileKeys []string,
	serializedSize uint64,
	statistic *indexpb.JobInfo,
	currentIndexVersion int32,
	indexStoreVersion int64,
) {
	key := taskKey{ClusterID: ClusterID, BuildID: buildID}
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	if info, ok := i.indexTasks[key]; ok {
		info.fileKeys = common.CloneStringList(fileKeys)
		info.serializedSize = serializedSize
		info.statistic = proto.Clone(statistic).(*indexpb.JobInfo)
		info.currentIndexVersion = currentIndexVersion
		info.indexStoreVersion = indexStoreVersion
		return
	}
}

func (i *IndexNode) deleteIndexTaskInfos(ctx context.Context, keys []taskKey) []*indexTaskInfo {
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	deleted := make([]*indexTaskInfo, 0, len(keys))
	for _, key := range keys {
		info, ok := i.indexTasks[key]
		if ok {
			deleted = append(deleted, info)
			delete(i.indexTasks, key)
			log.Ctx(ctx).Info("delete task infos",
				zap.String("cluster_id", key.ClusterID), zap.Int64("build_id", key.BuildID))
		}
	}
	return deleted
}

func (i *IndexNode) deleteAllIndexTasks() []*indexTaskInfo {
	i.stateLock.Lock()
	deletedTasks := i.indexTasks
	i.indexTasks = make(map[taskKey]*indexTaskInfo)
	i.stateLock.Unlock()

	deleted := make([]*indexTaskInfo, 0, len(deletedTasks))
	for _, info := range deletedTasks {
		deleted = append(deleted, info)
	}
	return deleted
}

type analysisTaskInfo struct {
	cancel                context.CancelFunc
	state                 commonpb.IndexState
	failReason            string
	centroidsFile         string
	segmentsOffsetMapping map[int64]string
	indexStoreVersion     int64
}

func (i *IndexNode) loadOrStoreAnalysisTask(clusterID string, taskID UniqueID, info *analysisTaskInfo) *analysisTaskInfo {
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	key := taskKey{ClusterID: clusterID, BuildID: taskID}
	oldInfo, ok := i.analysisTasks[key]
	if ok {
		return oldInfo
	}
	i.analysisTasks[key] = info
	return nil
}

func (i *IndexNode) loadAnalysisTaskState(clusterID string, taskID UniqueID) commonpb.IndexState {
	key := taskKey{ClusterID: clusterID, BuildID: taskID}
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	task, ok := i.analysisTasks[key]
	if !ok {
		return commonpb.IndexState_IndexStateNone
	}
	return task.state
}

func (i *IndexNode) storeAnalysisTaskState(clusterID string, taskID UniqueID, state commonpb.IndexState, failReason string) {
	key := taskKey{ClusterID: clusterID, BuildID: taskID}
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	if task, ok := i.analysisTasks[key]; ok {
		log.Info("IndexNode store analysis task state", zap.String("clusterID", clusterID), zap.Int64("taskID", taskID),
			zap.String("state", state.String()), zap.String("fail reason", failReason))
		task.state = state
		task.failReason = failReason
	}
}

func (i *IndexNode) foreachAnalysisTaskInfo(fn func(clusterID string, taskID UniqueID, info *analysisTaskInfo)) {
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	for key, info := range i.analysisTasks {
		fn(key.ClusterID, key.BuildID, info)
	}
}

func (i *IndexNode) getAnalysisTaskInfo(clusterID string, taskID UniqueID) *analysisTaskInfo {
	i.stateLock.Lock()
	defer i.stateLock.Unlock()

	return i.analysisTasks[taskKey{ClusterID: clusterID, BuildID: taskID}]
}

func (i *IndexNode) storeAnalysisStatistic(
	clusterID string,
	taskID UniqueID,
	centroidsFile string,
	segmentsOffsetMapping map[int64]string,
) {
	key := taskKey{ClusterID: clusterID, BuildID: taskID}
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	if info, ok := i.analysisTasks[key]; ok {
		info.centroidsFile = centroidsFile
		info.segmentsOffsetMapping = segmentsOffsetMapping
		return
	}
}

func (i *IndexNode) deleteAnalysisTaskInfos(ctx context.Context, keys []taskKey) []*analysisTaskInfo {
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	deleted := make([]*analysisTaskInfo, 0, len(keys))
	for _, key := range keys {
		info, ok := i.analysisTasks[key]
		if ok {
			deleted = append(deleted, info)
			delete(i.analysisTasks, key)
			log.Ctx(ctx).Info("delete analysis task infos",
				zap.String("clusterID", key.ClusterID), zap.Int64("taskID", key.BuildID))
		}
	}
	return deleted
}

func (i *IndexNode) deleteAllAnalysisTasks() []*analysisTaskInfo {
	i.stateLock.Lock()
	deletedTasks := i.analysisTasks
	i.analysisTasks = make(map[taskKey]*analysisTaskInfo)
	i.stateLock.Unlock()

	deleted := make([]*analysisTaskInfo, 0, len(deletedTasks))
	for _, info := range deletedTasks {
		deleted = append(deleted, info)
	}
	return deleted
}

func (i *IndexNode) hasInProgressTask() bool {
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	for _, info := range i.indexTasks {
		if info.state == commonpb.IndexState_InProgress {
			return true
		}
	}

	for _, info := range i.analysisTasks {
		if info.state == commonpb.IndexState_InProgress {
			return true
		}
	}
	return false
}

func (i *IndexNode) waitTaskFinish() {
	if !i.hasInProgressTask() {
		return
	}

	gracefulTimeout := &Params.IndexNodeCfg.GracefulStopTimeout
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(i.loopCtx, gracefulTimeout.GetAsDuration(time.Second))
	defer cancel()
	for {
		select {
		case <-ticker.C:
			if !i.hasInProgressTask() {
				return
			}
		case <-timeoutCtx.Done():
			log.Warn("timeout, the index node has some progress task")
			for _, info := range i.indexTasks {
				if info.state == commonpb.IndexState_InProgress {
					log.Warn("progress task", zap.Any("info", info))
				}
			}
			for _, info := range i.analysisTasks {
				if info.state == commonpb.IndexState_InProgress {
					log.Warn("progress task", zap.Any("info", info))
				}
			}
			return
		}
	}
}
