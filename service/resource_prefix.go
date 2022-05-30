package service

import (
	"github.com/recode-sh/recode/entities"
)

func prefixClusterResource(
	clusterNameSlug string,
) func(string) string {

	return func(resourceNameSlug string) string {
		if clusterNameSlug == entities.DefaultClusterName {
			return "recode-" + resourceNameSlug
		}

		return "recode-" + clusterNameSlug + "-" + resourceNameSlug
	}
}

func prefixDevEnvResource(
	clusterNameSlug string,
	devEnvNameSlug string,
) func(string) string {

	return func(resourceNameSlug string) string {
		if clusterNameSlug == entities.DefaultClusterName {
			return "recode-" + devEnvNameSlug + "-" + resourceNameSlug
		}

		return "recode-" + clusterNameSlug + "-" + devEnvNameSlug + "-" + resourceNameSlug
	}
}
