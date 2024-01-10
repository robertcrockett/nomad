package nomad

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/nomad/nomad/state"
	"github.com/hashicorp/nomad/nomad/structs"
)

func NewJobsEndpoint(s *Server, ctx *RPCContext) *Jobs {
	return &Jobs{
		srv:    s,
		ctx:    ctx,
		logger: s.logger.Named("jobs"),
	}
}

type Jobs struct {
	srv    *Server
	ctx    *RPCContext
	logger hclog.Logger
}

func (j *Jobs) Statuses(
	args *structs.JobsStatusesRequest,
	reply *structs.JobsStatusesResponse) error {
	// TODO: auth, rate limiting, etc...

	if reply.Jobs == nil {
		reply.Jobs = make(map[string]structs.JobStatus)
	}

	opts := blockingOptions{
		queryOpts: &args.QueryOptions,
		queryMeta: &reply.QueryMeta,
		run: func(ws memdb.WatchSet, state *state.StateStore) error {
			var idx uint64

			for _, j := range args.Jobs {
				ns := j.Namespace
				job, err := state.JobByID(ws, ns, j.ID)
				if err != nil {
					return err
				}
				if job == nil {
					continue
				}

				js := structs.JobStatus{
					ID:        j.ID,
					Namespace: j.Namespace,
				}
				js.Type = job.Type
				for _, tg := range job.TaskGroups {
					js.GroupCountSum += tg.Count
				}

				allocs, err := state.AllocsByJob(ws, ns, j.ID, false)
				if err != nil {
					return err
				}
				for _, a := range allocs {
					alloc := structs.JobStatusAlloc{
						ID:           a.ID,
						Group:        a.TaskGroup,
						ClientStatus: a.ClientStatus,
					}
					if a.DeploymentStatus != nil {
						alloc.DeploymentStatus.Canary = a.DeploymentStatus.Canary
						if a.DeploymentStatus.Healthy != nil {
							alloc.DeploymentStatus.Healthy = *a.DeploymentStatus.Healthy
						}
					}
					js.Allocs = append(js.Allocs, alloc)
					if a.ModifyIndex > idx {
						idx = a.ModifyIndex
					}
				}

				deploys, err := state.DeploymentsByJobID(ws, ns, j.ID, false)
				if err != nil {
					return err
				}
				for _, d := range deploys {
					if d.Active() {
						js.DeploymentID = d.ID
						break
					}
					if d.ModifyIndex > idx {
						idx = d.ModifyIndex
					}
				}

				nsid := fmt.Sprintf("%s@%s", j.ID, j.Namespace)
				reply.Jobs[nsid] = js
			}
			reply.Index = idx
			j.srv.setQueryMeta(&reply.QueryMeta)
			return nil

		}}
	return j.srv.blockingRPC(&opts)
}
