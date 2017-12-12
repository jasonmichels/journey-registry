package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jasonmichels/journey-registry/journey"
)

// CacheV The cached version, if it exists
type CacheV struct {
	Key      string
	Version  *journey.Version
	CachedAt int64
}

// Explorer Explorer server
type Explorer struct {
	VersionCache map[string]*CacheV
	AWS          *session.Session
	Bucket       string
}

// GetDependencies Get all the assets for your postcards being used on journey
func (exp *Explorer) GetDependencies(ctx context.Context, r *journey.Journey) (*journey.DependencyAssets, error) {
	var response journey.DependencyAssets
	var versions []*journey.Version

	var wg sync.WaitGroup
	wg.Add(len(r.Dependencies))
	c := make(chan *CacheV, len(r.Dependencies))

	svc := s3.New(exp.AWS)

	for k, v := range r.Dependencies {
		// k -- eg. widget-name
		// v -- eg. 1.0.2
		pathKey := k + "/" + v + "/journey-urls.json"
		bucket := exp.Bucket

		if ok := loadJourneyURLCachedVersion(exp.VersionCache, pathKey, c, &wg); ok == true {
			continue
		}

		input := &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(pathKey),
		}

		go loadJourneyURLFromS3(svc, input, c, &wg)
	}

	wg.Wait()
	close(c)

	for cachedVersion := range c {
		if cachedVersion != nil {
			exp.VersionCache[cachedVersion.Key] = cachedVersion
			versions = append(versions, cachedVersion.Version)
		}
	}

	response.Versions = versions

	return &response, nil
}

// loadJourneyUrlCachedVersion check for cached version before expensive trip to AWS S3 bucket
func loadJourneyURLCachedVersion(cache map[string]*CacheV, key string, c chan *CacheV, wg *sync.WaitGroup) bool {
	isCached := false

	if cached, ok := cache[key]; ok {
		elapsed := time.Since(time.Unix(cached.CachedAt, 0))

		if elapsed.Minutes() < 5.0 {
			wg.Done()
			c <- cached
			isCached = true
		}
	}

	return isCached
}

func loadJourneyURLFromS3(svc *s3.S3, input *s3.GetObjectInput, c chan *CacheV, wg *sync.WaitGroup) {
	// @TODO maybe use a channel of errors to capture and log errors while still allowing to continue
	defer wg.Done()

	var version journey.Version

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result, err := svc.GetObjectWithContext(ctx, input)
	if err != nil {
		return
	}
	defer result.Body.Close()

	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return
	}

	if err := json.Unmarshal(body, &version); err != nil {
		return
	}

	c <- &CacheV{Key: *input.Key, Version: &version, CachedAt: time.Now().Unix()}
}
