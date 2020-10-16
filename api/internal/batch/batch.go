package batch

import "math"

func ToBatchesWithTransformer(items []interface{}, batchSize int, batchTransformer func([]interface{}) interface{}) []interface{} {
	numBatches := int(math.Ceil(float64(len(items)) / float64(batchSize)))
	batches := make([]interface{}, numBatches)

	for batchNum := 0; batchNum < numBatches; batchNum++ {
		startIndex := batchNum * batchSize
		endIndex := int(math.Min(float64(len(items)), float64((batchNum+1)*batchSize)))

		batch := []interface{}{}
		for i := startIndex; i < endIndex; i++ {
			batch = append(batch, items[i])
		}

		batches[batchNum] = batchTransformer(batch)
	}

	return batches
}
