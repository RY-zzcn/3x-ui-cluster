package service

import (
	"time"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
)

type SlaveCertService struct{}

// GetCertsForSlave returns all certificates for a specific slave
func (s *SlaveCertService) GetCertsForSlave(slaveId int) ([]*model.SlaveCert, error) {
	db := database.GetDB()
	var certs []*model.SlaveCert
	err := db.Where("slave_id = ?", slaveId).Order("domain").Find(&certs).Error
	return certs, err
}

// GetAllCerts returns all certificates across all slaves
func (s *SlaveCertService) GetAllCerts() ([]*model.SlaveCert, error) {
	db := database.GetDB()
	var certs []*model.SlaveCert
	err := db.Order("slave_id, domain").Find(&certs).Error
	return certs, err
}

// UpsertCert inserts or updates a certificate record
func (s *SlaveCertService) UpsertCert(cert *model.SlaveCert) error {
	db := database.GetDB()
	cert.LastUpdated = time.Now().Unix()
	
	// Check if cert already exists
	var existing model.SlaveCert
	err := db.Where("slave_id = ? AND domain = ?", cert.SlaveId, cert.Domain).First(&existing).Error
	
	if err == nil {
		// Update existing
		cert.Id = existing.Id
		return db.Save(cert).Error
	}
	
	// Insert new
	return db.Create(cert).Error
}

// DeleteCert deletes a certificate by ID
func (s *SlaveCertService) DeleteCert(id int) error {
	db := database.GetDB()
	return db.Delete(&model.SlaveCert{}, id).Error
}

// DeleteCertsForSlave deletes all certificates for a slave
func (s *SlaveCertService) DeleteCertsForSlave(slaveId int) error {
	db := database.GetDB()
	return db.Where("slave_id = ?", slaveId).Delete(&model.SlaveCert{}).Error
}

// BatchUpsertCerts updates multiple certificates at once (for slave reporting)
func (s *SlaveCertService) BatchUpsertCerts(slaveId int, certs []model.SlaveCert) error {
	db := database.GetDB()
	
	// Start transaction
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// Process each cert
	for _, cert := range certs {
		cert.SlaveId = slaveId
		cert.LastUpdated = time.Now().Unix()
		
		var existing model.SlaveCert
		err := tx.Where("slave_id = ? AND domain = ?", slaveId, cert.Domain).First(&existing).Error
		
		if err == nil {
			// Update existing
			cert.Id = existing.Id
			if err := tx.Save(&cert).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else {
			// Create new
			if err := tx.Create(&cert).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	
	return tx.Commit().Error
}
