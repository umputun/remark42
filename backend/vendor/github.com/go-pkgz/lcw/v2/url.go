package lcw

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/redis/go-redis/v9"
)

// New parses uri and makes any of supported caches
// supported URIs:
//   - redis://<ip>:<port>?db=123&max_keys=10
//   - mem://lru?max_keys=10&max_cache_size=1024
//   - mem://expirable?ttl=30s&max_val_size=100
//   - nop://
func New[V any](uri string) (LoadingCache[V], error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse cache uri %s: %w", uri, err)
	}

	query := u.Query()
	opts, err := optionsFromQuery[V](query)
	if err != nil {
		return nil, fmt.Errorf("parse uri options %s: %w", uri, err)
	}

	switch u.Scheme {
	case "redis":
		redisOpts, e := redisOptionsFromURL(u)
		if e != nil {
			return nil, e
		}
		res, e := NewRedisCache(redis.NewClient(redisOpts), opts...)
		if e != nil {
			return nil, fmt.Errorf("make redis for %s: %w", uri, e)
		}
		return res, nil
	case "mem":
		switch u.Hostname() {
		case "lru":
			return NewLruCache[V](opts...)
		case "expirable":
			return NewExpirableCache[V](opts...)
		default:
			return nil, fmt.Errorf("unsupported mem cache type %s", u.Hostname())
		}
	case "nop":
		return NewNopCache[V](), nil
	}
	return nil, fmt.Errorf("unsupported cache type %s", u.Scheme)
}

func optionsFromQuery[V any](q url.Values) (opts []Option[V], err error) {
	errs := new(multierror.Error)
	o := NewOpts[V]()

	if v := q.Get("max_val_size"); v != "" {
		vv, e := strconv.Atoi(v)
		if e != nil {
			errs = multierror.Append(errs, fmt.Errorf("max_val_size query param %s: %w", v, e))
		} else {
			opts = append(opts, o.MaxValSize(vv))
		}
	}

	if v := q.Get("max_key_size"); v != "" {
		vv, e := strconv.Atoi(v)
		if e != nil {
			errs = multierror.Append(errs, fmt.Errorf("max_key_size query param %s: %w", v, e))
		} else {
			opts = append(opts, o.MaxKeySize(vv))
		}
	}

	if v := q.Get("max_keys"); v != "" {
		vv, e := strconv.Atoi(v)
		if e != nil {
			errs = multierror.Append(errs, fmt.Errorf("max_keys query param %s: %w", v, e))
		} else {
			opts = append(opts, o.MaxKeys(vv))
		}
	}

	if v := q.Get("max_cache_size"); v != "" {
		vv, e := strconv.ParseInt(v, 10, 64)
		if e != nil {
			errs = multierror.Append(errs, fmt.Errorf("max_cache_size query param %s: %w", v, e))
		} else {
			opts = append(opts, o.MaxCacheSize(vv))
		}
	}

	if v := q.Get("ttl"); v != "" {
		vv, e := time.ParseDuration(v)
		if e != nil {
			errs = multierror.Append(errs, fmt.Errorf("ttl query param %s: %w", v, e))
		} else {
			opts = append(opts, o.TTL(vv))
		}
	}

	return opts, errs.ErrorOrNil()
}

func redisOptionsFromURL(u *url.URL) (*redis.Options, error) {
	query := u.Query()

	db, err := strconv.Atoi(query.Get("db"))
	if err != nil {
		return nil, fmt.Errorf("db from %s: %w", u, err)
	}

	res := &redis.Options{
		Addr:     u.Hostname() + ":" + u.Port(),
		DB:       db,
		Password: query.Get("password"),
		Network:  query.Get("network"),
	}

	if dialTimeout, err := time.ParseDuration(query.Get("dial_timeout")); err == nil {
		res.DialTimeout = dialTimeout
	}

	if readTimeout, err := time.ParseDuration(query.Get("read_timeout")); err == nil {
		res.ReadTimeout = readTimeout
	}

	if writeTimeout, err := time.ParseDuration(query.Get("write_timeout")); err == nil {
		res.WriteTimeout = writeTimeout
	}

	return res, nil
}
