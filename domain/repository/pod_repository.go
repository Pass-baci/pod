package repository

import (
	"github.com/Pass-baci/pod/domain/model"
	"gorm.io/gorm"
)

type IPodRepository interface {
	// InitTable 初始化表
	InitTable() error
	// FindPodByID 根据ID查找Pod数据
	FindPodByID(id int64) (*model.Pod, error)
	// CreatePod 创建Pod
	CreatePod(pod *model.Pod) (int64, error)
	// DeletePodByID 根据ID删除Pod
	DeletePodByID(id int64) error
	// UpdatePod 修改Pod
	UpdatePod(pod *model.Pod) error
	// FindAll 查找所有Pod
	FindAll() ([]model.Pod, error)
}

func NewPodRepository(db *gorm.DB) IPodRepository {
	return &PodRepository{mysqlDB: db}
}

type PodRepository struct {
	mysqlDB *gorm.DB
}

func (p *PodRepository) InitTable() error {
	return p.mysqlDB.AutoMigrate(&model.Pod{}, &model.PodPort{}, &model.PodEnv{})
}

func (p *PodRepository) FindPodByID(id int64) (*model.Pod, error) {
	var res = &model.Pod{}
	return res, p.mysqlDB.Preload("PodEnv").Preload("PodPort").First(res, id).Error
}

func (p *PodRepository) CreatePod(pod *model.Pod) (int64, error) {
	return pod.ID, p.mysqlDB.Create(pod).Error
}

func (p *PodRepository) DeletePodByID(id int64) error {
	tx := p.mysqlDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if tx.Error != nil {
		return tx.Error
	}

	// 删除pod信息
	if err := p.mysqlDB.Where("id = ?", id).Delete(&model.Pod{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除pod环境变量信息
	if err := p.mysqlDB.Where("pod_id = ?", id).Delete(&model.PodEnv{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除pod端口信息
	if err := p.mysqlDB.Where("pod_id = ?", id).Delete(&model.PodPort{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (p *PodRepository) UpdatePod(pod *model.Pod) error {
	return p.mysqlDB.Model(pod).Updates(pod).Error
}

func (p *PodRepository) FindAll() ([]model.Pod, error) {
	var res = make([]model.Pod, 0)
	return res, p.mysqlDB.Find(&res).Error
}
