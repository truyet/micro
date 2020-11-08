package handler

import (
	"context"
	"time"

	"github.com/micro/micro/v3/platform/auth/namespace"
	pb "github.com/micro/micro/v3/proto/runtime"
	"github.com/micro/micro/v3/service/auth"
	"github.com/micro/micro/v3/service/errors"
	"github.com/micro/micro/v3/service/events"
	goevents "github.com/micro/micro/v3/service/events"
	log "github.com/micro/micro/v3/service/logger"
	"github.com/micro/micro/v3/service/runtime"
	gorun "github.com/micro/micro/v3/service/runtime"
)

type Runtime struct {
	Runtime gorun.Runtime
}

func setupServiceMeta(ctx context.Context, service *runtime.Service) {
	if service.Metadata == nil {
		service.Metadata = map[string]string{}
	}
	account, accOk := auth.AccountFromContext(ctx)
	if accOk {
		// Try to use the account name as it's more user friendly. If none, fall back to ID
		owner := account.Name
		if len(owner) == 0 {
			owner = account.ID
		}
		service.Metadata["owner"] = owner
		// This is a hack - we don't want vanilla `micro server` users where the auth is noop
		// to have long uuid as owners, so we put micro here - not great, not terrible.
		if auth.DefaultAuth.String() == "noop" {
			service.Metadata["owner"] = "micro"
		}
		service.Metadata["group"] = account.Issuer
	}
	service.Metadata["started"] = time.Now().Format(time.RFC3339)
}

func (r *Runtime) Read(ctx context.Context, req *pb.ReadRequest, rsp *pb.ReadResponse) error {
	// set defaults
	if req.Options == nil {
		req.Options = &pb.ReadOptions{}
	}
	if len(req.Options.Namespace) == 0 {
		req.Options.Namespace = namespace.DefaultNamespace
	}

	// authorize the request
	if err := namespace.Authorize(ctx, req.Options.Namespace); err == namespace.ErrForbidden {
		return errors.Forbidden("runtime.Runtime.Read", err.Error())
	} else if err == namespace.ErrUnauthorized {
		return errors.Unauthorized("runtime.Runtime.Read", err.Error())
	} else if err != nil {
		return errors.InternalServerError("runtime.Runtime.Read", err.Error())
	}

	// lookup the services
	options := toReadOptions(ctx, req.Options)
	services, err := r.Runtime.Read(options...)
	if err != nil {
		return errors.InternalServerError("runtime.Runtime.Read", err.Error())
	}

	// serialize the response
	for _, service := range services {
		rsp.Services = append(rsp.Services, toProto(service))
	}

	return nil
}

func (r *Runtime) Logs(ctx context.Context, req *pb.LogsRequest, stream pb.Runtime_LogsStream) error {
	// set defaults
	if req.Options == nil {
		req.Options = &pb.LogsOptions{}
	}
	if len(req.Options.Namespace) == 0 {
		req.Options.Namespace = namespace.DefaultNamespace
	}

	// authorize the request
	if err := namespace.Authorize(ctx, req.Options.Namespace); err == namespace.ErrForbidden {
		return errors.Forbidden("runtime.Runtime.Logs", err.Error())
	} else if err == namespace.ErrUnauthorized {
		return errors.Unauthorized("runtime.Runtime.Logs", err.Error())
	} else if err != nil {
		return errors.InternalServerError("runtime.Runtime.Logs", err.Error())
	}

	opts := toLogsOptions(ctx, req.Options)

	// options passed in the request
	if req.GetCount() > 0 {
		opts = append(opts, runtime.LogsCount(req.GetCount()))
	}
	if req.GetStream() {
		opts = append(opts, runtime.LogsStream(req.GetStream()))
	}

	logStream, err := r.Runtime.Logs(&runtime.Service{
		Name: req.GetService(),
	}, opts...)
	if err != nil {
		return err
	}
	defer logStream.Stop()
	defer stream.Close()

	recordChan := logStream.Chan()
	for {
		select {
		case record, ok := <-recordChan:
			if !ok {
				return logStream.Error()
			}
			// send record
			if err := stream.Send(&pb.LogRecord{
				//Timestamp: record.Timestamp.Unix(),
				Metadata: record.Metadata,
				Message:  record.Message,
			}); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// Create a resource
func (r *Runtime) Create(ctx context.Context, req *pb.CreateRequest, rsp *pb.CreateResponse) error {

	// validate the request
	if req.Resource == nil || (req.Resource.Namespace == nil && req.Resource.Networkpolicy == nil && req.Resource.Resourcequota == nil && req.Resource.Service == nil) {
		return errors.BadRequest("runtime.Runtime.Create", "blank resource")
	}

	// set defaults
	if req.Options == nil {
		req.Options = &pb.CreateOptions{}
	}
	if len(req.Options.Namespace) == 0 {
		req.Options.Namespace = namespace.DefaultNamespace
	}

	// authorize the request
	if err := namespace.Authorize(ctx, req.Options.Namespace); err == namespace.ErrForbidden {
		return errors.Forbidden("runtime.Runtime.Create", err.Error())
	} else if err == namespace.ErrUnauthorized {
		return errors.Unauthorized("runtime.Runtime.Create", err.Error())
	} else if err != nil {
		return errors.InternalServerError("runtime.Runtime.Create", err.Error())
	}

	// Handle the different possible types of resource
	switch {
	case req.Resource.Namespace != nil:
		ns, err := gorun.NewNamespace(req.Resource.Namespace.Name)
		if err != nil {
			return err
		}

		if err := r.Runtime.Create(ns, gorun.CreateNamespace(req.Resource.Namespace.Name)); err != nil {
			return err
		}

		ev := &runtime.EventResourcePayload{
			Type:      runtime.EventNamespaceCreated,
			Namespace: ns.Name,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      runtime.EventNamespaceCreated,
			"namespace": ns.Name,
		}))

	case req.Resource.Networkpolicy != nil:
		np, err := gorun.NewNetworkPolicy(req.Resource.Networkpolicy.Name, req.Resource.Networkpolicy.Namespace, req.Resource.Networkpolicy.Allowedlabels)
		if err != nil {
			return err
		}

		if err := r.Runtime.Create(np, gorun.CreateNamespace(req.Resource.Networkpolicy.Namespace)); err != nil {
			return err
		}

		ev := &runtime.EventResourcePayload{
			Type:          runtime.EventNetworkPolicyCreated,
			Name:          np.Name,
			Namespace:     np.Namespace,
			NetworkPolicy: np,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      ev.Type,
			"namespace": ev.Namespace,
		}))

	case req.Resource.Resourcequota != nil:
		rq, err := gorun.NewResourceQuota(
			req.Resource.Resourcequota.Name,
			req.Resource.Resourcequota.Namespace,
			&gorun.Resources{
				CPU:  int(req.Resource.Resourcequota.Requests.CPU),
				Disk: int(req.Resource.Resourcequota.Requests.EphemeralStorage),
				Mem:  int(req.Resource.Resourcequota.Requests.Memory),
			},
			&gorun.Resources{
				CPU:  int(req.Resource.Resourcequota.Limits.CPU),
				Disk: int(req.Resource.Resourcequota.Limits.EphemeralStorage),
				Mem:  int(req.Resource.Resourcequota.Limits.Memory),
			},
		)
		if err != nil {
			return err
		}

		if err := r.Runtime.Create(rq, gorun.CreateNamespace(req.Resource.Resourcequota.Namespace)); err != nil {
			return err
		}

		ev := &runtime.EventResourcePayload{
			Type:      runtime.EventResourceQuotaCreated,
			Name:      rq.Name,
			Namespace: rq.Namespace,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      ev.Type,
			"namespace": ev.Namespace,
		}))

	case req.Resource.Service != nil:

		// create the service
		service := toService(req.Resource.Service)
		setupServiceMeta(ctx, service)

		options := toCreateOptions(ctx, req.Options)

		log.Infof("Creating service %s version %s source %s", service.Name, service.Version, service.Source)
		if err := r.Runtime.Create(service, options...); err != nil {
			return errors.InternalServerError("runtime.Runtime.Create", err.Error())
		}

		// publish the create event
		ev := &runtime.EventPayload{
			Service:   service,
			Namespace: req.Options.Namespace,
			Type:      runtime.EventServiceCreated,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      runtime.EventServiceCreated,
			"namespace": req.Options.Namespace,
		}))

	default:
		return nil
	}
}

// Delete a resource
func (r *Runtime) Delete(ctx context.Context, req *pb.DeleteRequest, rsp *pb.DeleteResponse) error {

	// validate the request
	if req.Resource == nil || (req.Resource.Namespace == nil && req.Resource.Networkpolicy == nil && req.Resource.Resourcequota == nil && req.Resource.Service == nil) {
		return errors.BadRequest("runtime.Runtime.Delete", "blank resource")
	}

	// set defaults
	if req.Options == nil {
		req.Options = &pb.DeleteOptions{}
	}
	if len(req.Options.Namespace) == 0 {
		req.Options.Namespace = namespace.DefaultNamespace
	}

	// authorize the request
	if err := namespace.Authorize(ctx, req.Options.Namespace); err == namespace.ErrForbidden {
		return errors.Forbidden("runtime.Runtime.Delete", err.Error())
	} else if err == namespace.ErrUnauthorized {
		return errors.Unauthorized("runtime.Runtime.Delete", err.Error())
	} else if err != nil {
		return errors.InternalServerError("runtime.Runtime.Delete", err.Error())
	}

	// Handle the different possible types of resource
	switch {
	case req.Resource.Namespace != nil:
		ns, err := gorun.NewNamespace(req.Resource.Namespace.Name)
		if err != nil {
			return err
		}

		if err := r.Runtime.Delete(ns, gorun.DeleteNamespace(req.Resource.Namespace.Name)); err != nil {
			return err
		}

		ev := &runtime.EventResourcePayload{
			Type:      runtime.EventNamespaceDeleted,
			Namespace: ns.Name,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      runtime.EventNamespaceDeleted,
			"namespace": ns.Name,
		}))

	case req.Resource.Networkpolicy != nil:
		np, err := gorun.NewNetworkPolicy(req.Resource.Networkpolicy.Name, req.Resource.Networkpolicy.Namespace, req.Resource.Networkpolicy.Allowedlabels)
		if err != nil {
			return err
		}

		if err := r.Runtime.Delete(np, gorun.DeleteNamespace(req.Resource.Networkpolicy.Namespace)); err != nil {
			return err
		}

		ev := &runtime.EventResourcePayload{
			Type:          runtime.EventNetworkPolicyDeleted,
			Name:          np.Name,
			Namespace:     np.Namespace,
			NetworkPolicy: np,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      ev.Type,
			"namespace": ev.Namespace,
		}))

	case req.Resource.Resourcequota != nil:
		rq, err := gorun.NewResourceQuota(
			req.Resource.Resourcequota.Name,
			req.Resource.Resourcequota.Namespace,
			&gorun.Resources{
				CPU:  int(req.Resource.Resourcequota.Requests.CPU),
				Disk: int(req.Resource.Resourcequota.Requests.EphemeralStorage),
				Mem:  int(req.Resource.Resourcequota.Requests.Memory),
			},
			&gorun.Resources{
				CPU:  int(req.Resource.Resourcequota.Limits.CPU),
				Disk: int(req.Resource.Resourcequota.Limits.EphemeralStorage),
				Mem:  int(req.Resource.Resourcequota.Limits.Memory),
			},
		)
		if err != nil {
			return err
		}

		if err := r.Runtime.Delete(rq, gorun.DeleteNamespace(req.Resource.Resourcequota.Namespace)); err != nil {
			return err
		}

		ev := &runtime.EventResourcePayload{
			Type:      runtime.EventResourceQuotaDeleted,
			Name:      rq.Name,
			Namespace: rq.Namespace,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      ev.Type,
			"namespace": ev.Namespace,
		}))

	case req.Resource.Service != nil:

		// delete the service
		service := toService(req.Resource.Service)
		options := toDeleteOptions(ctx, req.Options)

		log.Infof("Deleting service %s version %s source %s", service.Name, service.Version, service.Source)
		if err := r.Runtime.Delete(service, options...); err != nil {
			return errors.InternalServerError("runtime.Runtime.Delete", err.Error())
		}

		// publish the delete event
		ev := &runtime.EventPayload{
			Type:      runtime.EventServiceDeleted,
			Namespace: req.Options.Namespace,
			Service:   service,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      runtime.EventServiceDeleted,
			"namespace": req.Options.Namespace,
		}))

	default:
		return nil
	}
}

// Update a resource
func (r *Runtime) Update(ctx context.Context, req *pb.UpdateRequest, rsp *pb.UpdateResponse) error {

	// validate the request
	if req.Resource == nil || (req.Resource.Namespace == nil && req.Resource.Networkpolicy == nil && req.Resource.Resourcequota == nil && req.Resource.Service == nil) {
		return errors.BadRequest("runtime.Runtime.Update", "blank resource")
	}

	// set defaults
	if req.Options == nil {
		req.Options = &pb.UpdateOptions{}
	}
	if len(req.Options.Namespace) == 0 {
		req.Options.Namespace = namespace.DefaultNamespace
	}

	// authorize the request
	if err := namespace.Authorize(ctx, req.Options.Namespace); err == namespace.ErrForbidden {
		return errors.Forbidden("runtime.Runtime.Update", err.Error())
	} else if err == namespace.ErrUnauthorized {
		return errors.Unauthorized("runtime.Runtime.Update", err.Error())
	} else if err != nil {
		return errors.InternalServerError("runtime.Runtime.Update", err.Error())
	}

	// Handle the different possible types of resource
	switch {
	case req.Resource.Namespace != nil:
		// No updates to namespace
		return nil

	case req.Resource.Networkpolicy != nil:
		np, err := gorun.NewNetworkPolicy(req.Resource.Networkpolicy.Name, req.Resource.Networkpolicy.Namespace, req.Resource.Networkpolicy.Allowedlabels)
		if err != nil {
			return err
		}

		if err := r.Runtime.Update(np, gorun.UpdateNamespace(req.Resource.Networkpolicy.Namespace)); err != nil {
			return err
		}

		ev := &runtime.EventResourcePayload{
			Type:          runtime.EventNetworkPolicyUpdated,
			Name:          np.Name,
			Namespace:     np.Namespace,
			NetworkPolicy: np,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      ev.Type,
			"namespace": ev.Namespace,
		}))

	case req.Resource.Resourcequota != nil:
		rq, err := gorun.NewResourceQuota(
			req.Resource.Resourcequota.Name,
			req.Resource.Resourcequota.Namespace,
			&gorun.Resources{
				CPU:  int(req.Resource.Resourcequota.Requests.CPU),
				Disk: int(req.Resource.Resourcequota.Requests.EphemeralStorage),
				Mem:  int(req.Resource.Resourcequota.Requests.Memory),
			},
			&gorun.Resources{
				CPU:  int(req.Resource.Resourcequota.Limits.CPU),
				Disk: int(req.Resource.Resourcequota.Limits.EphemeralStorage),
				Mem:  int(req.Resource.Resourcequota.Limits.Memory),
			},
		)
		if err != nil {
			return err
		}

		if err := r.Runtime.Update(rq, gorun.UpdateNamespace(req.Resource.Resourcequota.Namespace)); err != nil {
			return err
		}

		ev := &runtime.EventResourcePayload{
			Type:      runtime.EventResourceQuotaUpdated,
			Name:      rq.Name,
			Namespace: rq.Namespace,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      ev.Type,
			"namespace": ev.Namespace,
		}))

	case req.Resource.Service != nil:

		service := toService(req.Resource.Service)
		setupServiceMeta(ctx, service)

		options := toUpdateOptions(ctx, req.Options)

		log.Infof("Updating service %s version %s source %s", service.Name, service.Version, service.Source)

		if err := r.Runtime.Update(service, options...); err != nil {
			return errors.InternalServerError("runtime.Runtime.Update", err.Error())
		}

		// publish the update event
		ev := &runtime.EventPayload{
			Service:   service,
			Namespace: req.Options.Namespace,
			Type:      runtime.EventServiceUpdated,
		}

		return events.Publish(runtime.EventTopic, ev, goevents.WithMetadata(map[string]string{
			"type":      runtime.EventServiceUpdated,
			"namespace": req.Options.Namespace,
		}))

	default:
		return nil
	}
}
