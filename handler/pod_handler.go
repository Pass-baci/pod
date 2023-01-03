package handler

import (
	"context"
	"github.com/Pass-baci/common"
	"github.com/Pass-baci/pod/domain/model"
	"github.com/Pass-baci/pod/domain/service"
	"github.com/Pass-baci/pod/proto/pod"
	"strconv"
)

type podHandle struct {
	podDataService service.IPodDataService
}

func NewPodHandle(podDataService service.IPodDataService) *podHandle {
	return &podHandle{podDataService: podDataService}
}

func (p *podHandle) AddPod(ctx context.Context, req *pod.PodInfo, rsp *pod.Response) error {
	var err error

	podModel := &model.Pod{}
	if err = common.SwapTo(req, podModel); err != nil {
		common.Errorf("Handle AddPod err: %s \n", err.Error())
		rsp.Msg = "Handle AddPod SwapTo Err"
		return err
	}

	if err = p.podDataService.CreateToK8s(req); err != nil {
		common.Errorf("Handle AddPod-CreateToK8s err: %s \n", err.Error())
		rsp.Msg = "Handle AddPod CreateToK8s Err"
		return err
	}

	var id int64
	if id, err = p.podDataService.AddPod(podModel); err != nil {
		common.Errorf("Handle AddPod-AddPod err: %s \n", err.Error())
		rsp.Msg = "Handle AddPod AddPod Err"
		return err
	}

	rsp.Msg = strconv.FormatInt(id, 10)
	common.Infof("Handle AddPod Success id=%d", id)
	return nil
}

func (p *podHandle) DeletePod(ctx context.Context, req *pod.PodId, rsp *pod.Response) error {
	podInfo, err := p.podDataService.FindPodByID(req.Id)
	if err != nil {
		common.Errorf("Handle DeletePod-FindPodByID err: %s \n", err.Error())
		rsp.Msg = "Handle DeletePod-FindPodByID Err"
		return err
	}

	if err = p.podDataService.DeleteFromK8s(podInfo); err != nil {
		common.Errorf("Handle DeletePod-DeleteFromK8s err: %s \n", err.Error())
		rsp.Msg = "Handle DeletePod-DeleteFromK8s Err"
		return err
	}

	if err = p.podDataService.DeletePod(req.Id); err != nil {
		common.Errorf("Handle DeletePod-DeletePod err: %s \n", err.Error())
		rsp.Msg = "Handle DeletePod-DeletePod Err"
		return err
	}

	rsp.Msg = "删除pod成功"
	common.Info("Handle DeletePod Success id")
	return nil
}

func (p *podHandle) FindPodByID(ctx context.Context, req *pod.PodId, rsp *pod.PodInfo) error {
	podInfo, err := p.podDataService.FindPodByID(req.Id)
	if err != nil {
		common.Errorf("Handle FindPodByID-FindPodByID err: %s \n", err.Error())
		return err
	}

	if err = common.SwapTo(podInfo, rsp); err != nil {
		common.Errorf("Handle FindPodByID-SwapTo err: %s \n", err.Error())
		return err
	}

	return nil
}

func (p *podHandle) UpdatePod(ctx context.Context, req *pod.PodInfo, rsp *pod.Response) error {
	var err error

	var podModel = &model.Pod{}
	if err = common.SwapTo(req, podModel); err != nil {
		common.Errorf("Handle UpdatePod-SwapTo err: %s \n", err.Error())
		rsp.Msg = "Handle UpdatePod-SwapTo Err"
		return err
	}

	if err = p.podDataService.UpdateToK8s(req); err != nil {
		common.Errorf("Handle UpdatePod-UpdateToK8s err: %s \n", err.Error())
		rsp.Msg = "Handle UpdatePod-UpdateToK8s Err"
		return err
	}

	if err = p.podDataService.UpdatePod(podModel); err != nil {
		common.Errorf("Handle UpdatePod-UpdatePod err: %s \n", err.Error())
		rsp.Msg = "Handle UpdatePod-UpdatePod Err"
		return err
	}

	return nil
}

func (p *podHandle) FindAllPod(ctx context.Context, req *pod.FindAll, rsp *pod.AllPod) error {
	allPod, err := p.podDataService.FindAllPod()
	if err != nil {
		common.Errorf("Handle FindAllPod-FindAllPod err: %s \n", err.Error())
		return err
	}

	for _, podInfo := range allPod {
		var temp = &pod.PodInfo{}
		if err = common.SwapTo(podInfo, temp); err != nil {
			common.Errorf("Handle FindAllPod-SwapTo err: %s \n", err.Error())
			return err
		}
		rsp.PodInfo = append(rsp.PodInfo, temp)
	}

	return nil
}
