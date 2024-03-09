package psbotfunc

import "go.uber.org/zap"

const (
	NOISE     = false
	CLUSTERED = true
)

type Cluster []DetectedText

var l *zap.Logger

func Clusterize(objects []DetectedText, minPts int, eps float64, logger *zap.Logger) []Cluster {
	l = logger
	clusters := make([]Cluster, 0)
	visited := make(map[string]bool)
	for _, point := range objects {
		if v, isVisited := visited[point.GetID()]; !v || !isVisited {
			neighbours := findNeighbours(point, objects, eps)
			if len(neighbours)+1 >= minPts {
				cluster := Cluster{}
				clusters = append(clusters, expandCluster(point, cluster, neighbours, objects, visited, minPts, eps))
			} else {
				visited[point.GetID()] = NOISE
			}
		}
		// l.Debug(point.ID, zap.Any("clusters", clusters))
	}
	// l.Debug("finish", zap.Any("clusters", clusters))
	l.Info("clustering finish.")
	return clusters
}

// Finds the neighbours from given array
// depends on Eps variable, which determines
// the distance limit from the point
func findNeighbours(point DetectedText, points []DetectedText, eps float64) []DetectedText {
	neighbours := make([]DetectedText, 0)
	// neighbours = append(neighbours, point)
	for _, potNeigb := range points {
		if point.GetID() != potNeigb.GetID() && potNeigb.Distance(point) <= eps {
			neighbours = append(neighbours, potNeigb)
		}
	}
	// l.Debug("neigher", zap.String(point.ID, point.Text), zap.Any("neighers", neighbours))
	return neighbours
}

// Try to expand existing clutser
func expandCluster(point DetectedText, cluster Cluster, neighbours, points []DetectedText, visited map[string]bool, minPts int, eps float64) Cluster {
	cluster = append(cluster, point)
	visited[point.GetID()] = CLUSTERED
	seed := make([]DetectedText, len(neighbours))
	copy(seed, neighbours)
	index := 0
	length := len(seed)
	for index < length {
		point := seed[index]
		pointState, isVisited := visited[point.GetID()]
		if !isVisited {
			currentNeighbours := findNeighbours(point, points, eps)
			if len(currentNeighbours)+1 >= minPts {
				visited[point.GetID()] = CLUSTERED
				seed = merge(seed, currentNeighbours, visited)
			}
		}

		if isVisited && !pointState {
			visited[point.GetID()] = CLUSTERED
			cluster = append(cluster, point)
		}

		length = len(seed)
		// l.Debug(point.GetID()+":"+point.Text, zap.Int("ind", index), zap.Any("seed", seed))
		index++
	}
	cluster = merge(cluster, seed, visited)
	// l.Debug("expand", zap.Any("cluster", cluster))
	for _, p := range cluster {
		visited[p.GetID()] = CLUSTERED
	}
	return cluster
}

func merge(one []DetectedText, two []DetectedText, visited map[string]bool) []DetectedText {
	mergeMap := make(map[string]DetectedText)
	putAll(mergeMap, one)
	putAll(mergeMap, two)
	merged := make([]DetectedText, 0)
	for _, val := range mergeMap {
		// visited[val.GetID()] = CLUSTERED
		merged = append(merged, val)
	}

	return merged
}

// Function to add all values from list to map
// map keys is then the unique collecton from list
func putAll(m map[string]DetectedText, list []DetectedText) {
	for _, val := range list {
		m[val.GetID()] = val
	}
}
