package api

import (
	"cf/api/resources"
	"cf/configuration"
	"cf/errors"
	"cf/models"
	"cf/net"
	"fmt"
	"net/url"
	"strings"
)

type RouteRepository interface {
	ListRoutes(cb func(models.Route) bool) (apiErr error)
	FindByHostAndDomain(host string, domain models.DomainFields) (route models.Route, apiErr error)
	Create(host, domainGuid string) (createdRoute models.Route, apiErr error)
	CreateInSpace(host, domainGuid, spaceGuid string) (createdRoute models.Route, apiErr error)
	Bind(routeGuid, appGuid string) (apiErr error)
	Unbind(routeGuid, appGuid string) (apiErr error)
	Delete(routeGuid string) (apiErr error)
}

type CloudControllerRouteRepository struct {
	config  configuration.Reader
	gateway net.Gateway
}

func NewCloudControllerRouteRepository(config configuration.Reader, gateway net.Gateway) (repo CloudControllerRouteRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerRouteRepository) ListRoutes(cb func(models.Route) bool) (apiErr error) {
	return repo.gateway.ListPaginatedResources(
		repo.config.ApiEndpoint(),
		fmt.Sprintf("/v2/spaces/%s/routes?inline-relations-depth=1", repo.config.SpaceFields().Guid),
		resources.RouteResource{},
		func(resource interface{}) bool {
			return cb(resource.(resources.RouteResource).ToModel())
		})
}

func (repo CloudControllerRouteRepository) FindByHostAndDomain(host string, domain models.DomainFields) (route models.Route, apiErr error) {
	found := false
	apiErr = repo.gateway.ListPaginatedResources(
		repo.config.ApiEndpoint(),
		fmt.Sprintf("/v2/routes?inline-relations-depth=1&q=%s", url.QueryEscape("host:"+host+";domain_guid:"+domain.Guid)),
		resources.RouteResource{},
		func(resource interface{}) bool {
			route = resource.(resources.RouteResource).ToModel()
			found = true
			return false
		})

	if apiErr == nil && !found {
		apiErr = errors.NewModelNotFoundError("Route", host)
	}

	return
}

func (repo CloudControllerRouteRepository) Create(host, domainGuid string) (createdRoute models.Route, apiErr error) {
	return repo.CreateInSpace(host, domainGuid, repo.config.SpaceFields().Guid)
}

func (repo CloudControllerRouteRepository) CreateInSpace(host, domainGuid, spaceGuid string) (createdRoute models.Route, apiErr error) {
	path := fmt.Sprintf("%s/v2/routes?inline-relations-depth=1", repo.config.ApiEndpoint())
	data := fmt.Sprintf(`{"host":"%s","domain_guid":"%s","space_guid":"%s"}`, host, domainGuid, spaceGuid)

	resource := new(resources.RouteResource)
	apiErr = repo.gateway.CreateResource(path, strings.NewReader(data), resource)
	if apiErr != nil {
		return
	}

	createdRoute = resource.ToModel()
	return
}

func (repo CloudControllerRouteRepository) Bind(routeGuid, appGuid string) (apiErr error) {
	path := fmt.Sprintf("%s/v2/apps/%s/routes/%s", repo.config.ApiEndpoint(), appGuid, routeGuid)
	return repo.gateway.UpdateResource(path, nil)
}

func (repo CloudControllerRouteRepository) Unbind(routeGuid, appGuid string) (apiErr error) {
	path := fmt.Sprintf("%s/v2/apps/%s/routes/%s", repo.config.ApiEndpoint(), appGuid, routeGuid)
	return repo.gateway.DeleteResource(path)
}

func (repo CloudControllerRouteRepository) Delete(routeGuid string) (apiErr error) {
	path := fmt.Sprintf("%s/v2/routes/%s", repo.config.ApiEndpoint(), routeGuid)
	return repo.gateway.DeleteResource(path)
}
