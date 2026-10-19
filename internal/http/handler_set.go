package http

import (
	"html/template"

	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
)

// HandlerSet berisi seluruh HTTP handler yang sudah tersedia.
//
// HandlerSet hanya menjadi container hasil construction.
// Ia tidak menjalankan routing, middleware, atau business logic.
type HandlerSet struct {
	Authentication    *AuthenticationHandler
	LoginPage         *LoginPageHandler
	Dashboard         *DashboardPageHandler
	AcademicYear      *AcademicYearPageHandler
	Class             *ClassPageHandler
	Student           *StudentPageHandler
	StudentPromotion  *StudentPromotionPageHandler
	ClassTransfer     *ClassTransferPageHandler
	TeacherAssignment *TeacherAssignmentPageHandler
	Users             *UserPageHandler
}

// NewHandlerSet membangun seluruh HTTP handler dari dependency yang
// disediakan oleh composition root.
func NewHandlerSet(
	services service.ServiceSet,
	securitySet security.SecuritySet,
	templates *template.Template,
) (HandlerSet, error) {
	authentication, err := NewAuthenticationHandler(
		services.Authentication,
		securitySet.SessionExpiration,
		securitySet.SessionCookie,
	)
	if err != nil {
		return HandlerSet{}, err
	}

	loginPage, err := NewLoginPageHandler(templates)
	if err != nil {
		return HandlerSet{}, err
	}

	dashboard, err := NewDashboardPageHandler(templates)
	if err != nil {
		return HandlerSet{}, err
	}

	academicYear, err := NewAcademicYearPageHandler(services.AcademicYear, templates)
	if err != nil {
		return HandlerSet{}, err
	}

	classHandler, err := NewClassPageHandler(services.Class, services.AcademicYear, templates)
	if err != nil {
		return HandlerSet{}, err
	}
	var studentHandler *StudentPageHandler
	if services.Student != nil && services.AcademicYear != nil && services.Class != nil && services.User != nil {
		studentHandler, err = NewStudentPageHandler(services.Student, services.AcademicYear, services.Class, services.User, templates)
		if err != nil {
			return HandlerSet{}, err
		}
	}
	var studentPromotion *StudentPromotionPageHandler
	if services.StudentPromotion != nil && services.AcademicYear != nil {
		studentPromotion, err = NewStudentPromotionPageHandler(services.StudentPromotion, services.AcademicYear, templates)
		if err != nil {
			return HandlerSet{}, err
		}
	}
	var classTransfer *ClassTransferPageHandler
	if services.ClassTransfer != nil {
		classTransfer, err = NewClassTransferPageHandler(services.ClassTransfer, templates)
		if err != nil {
			return HandlerSet{}, err
		}
	}

	var teacherAssignment *TeacherAssignmentPageHandler
	if services.ClassAssignment != nil && services.User != nil && services.Class != nil && services.AcademicYear != nil {
		teacherAssignment, err = NewTeacherAssignmentPageHandler(services.ClassAssignment, services.User, services.Class, services.AcademicYear, templates)
		if err != nil {
			return HandlerSet{}, err
		}
	}
	var userPage *UserPageHandler
	if services.User != nil && services.Student != nil && services.AcademicYear != nil && services.Class != nil && securitySet.PasswordHasher != nil {
		userPage, err = NewUserPageHandler(services.User, services.Student, services.AcademicYear, services.Class, securitySet.PasswordHasher, templates)
		if err != nil {
			return HandlerSet{}, err
		}
	}

	return HandlerSet{
		Authentication:    authentication,
		LoginPage:         loginPage,
		Dashboard:         dashboard,
		AcademicYear:      academicYear,
		Class:             classHandler,
		Student:           studentHandler,
		StudentPromotion:  studentPromotion,
		ClassTransfer:     classTransfer,
		TeacherAssignment: teacherAssignment,
		Users:             userPage,
	}, nil
}
