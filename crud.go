package db

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type basicCRUD struct {
}

type PaginationResult struct {
	Offset int
	Total  int64
	Limit  int
}

type FileUploader func(field string, file multipart.File, fileHeader *multipart.FileHeader) (string, string)

func (crud basicCRUD) FindByField(field string, value interface{}, model interface{}) error {
	o := orm.NewOrm()

	qs := o.QueryTable(model)
	err := qs.Filter(field, value).One(model)
	if err != nil {
		return err
	}

	return err
}

func (crud basicCRUD) FindAllByField(field string, value interface{}, model interface{}, entityList interface{}) error {
	o := orm.NewOrm()

	qs := o.QueryTable(model)
	_, err := qs.Filter(field, value).All(entityList)
	if err != nil {
		return err
	}

	return err
}

func (crud basicCRUD) Get(id int64, model interface{}) error {
	return crud.FindByField("id", id, model)
}

func (crud basicCRUD) Delete(model interface{}) (int64, error) {
	o := orm.NewOrm()
	count, err := o.Delete(model)
	return count, err
}

func (crud basicCRUD) Create(model interface{}) (int64, error) {
	o := orm.NewOrm()
	id, err := o.Insert(model)
	return id, err
}

func (crud basicCRUD) Save(model interface{}) (int64, error) {
	o := orm.NewOrm()
	count, err := o.Update(model)
	return count, err
}

func (crud basicCRUD) List(model interface{}) orm.QuerySeter {
	o := orm.NewOrm()
	return o.QueryTable(model)
}

func (crud basicCRUD) Paginate(query orm.QuerySeter, result interface{}, page int, perPage int) (int64, error) {
	offset := (page - 1) * perPage

	_ = offset
	//fmt.Println("offset:", offset, query)
	count, err := query.Offset(offset).Limit(perPage).All(result)

	return count, err
}

func (crud basicCRUD) FetchByQuery(model interface{}, getParams url.Values, result interface{}) (PaginationResult, error) {
	query := crud.List(model)

	presult := PaginationResult{Offset: 0}

	///

	filterBy := getParams.Get("filter")
	if filterBy != "" {
		parts := strings.Split(filterBy, "__")

		key := parts[0] + "__" + parts[1]
		//op := parts[1]

		value, _ := url.QueryUnescape(parts[2])

		query = query.Filter(key, value)
	}

	totalCount, err := query.Count()

	presult.Total = totalCount

	//
	offset := getParams.Get("offset")

	if offset != "" {
		offsetI, err := strconv.Atoi(offset)
		if err == nil {
			query = query.Offset(offsetI)
			presult.Offset = offsetI
		}
	}

	limit := getParams.Get("limit")
	limitI := 10
	if limit != "" {
		limitI, _ = strconv.Atoi(limit)
	}

	//
	page := getParams.Get("page")

	if page != "" {
		pageI, err := strconv.Atoi(page)
		if err == nil {
			offsetI := (pageI - 1) * limitI
			query = query.Offset(offsetI)
			presult.Offset = offsetI
		}
	}

	query = query.Limit(limitI)

	orderBy := getParams.Get("order")
	if orderBy == "" {
		orderBy = "-Id"
	}

	query = query.OrderBy(orderBy)

	presult.Limit = limitI

	query.All(result)

	return presult, err

}

func SetModelField(model reflect.Value, idx int, value string) {
	rfield := model.Field(idx)

	switch rfield.Kind() {
	case reflect.Int:
		ivalue, err := strconv.Atoi(value)
		if err == nil {
			rfield.SetInt(int64(ivalue))
		}

	case reflect.Int64:
		ivalue, err := strconv.Atoi(value)
		if err == nil {
			rfield.SetInt(int64(ivalue))
		}

	case reflect.Int32:
		ivalue, err := strconv.Atoi(value)
		if err == nil {
			rfield.SetInt(int64(ivalue))
		}

	case reflect.Int16:
		ivalue, err := strconv.Atoi(value)
		if err == nil {
			rfield.SetInt(int64(ivalue))
		}

	case reflect.Float32:
		fvalue, err := strconv.ParseFloat(value, 32)
		if err == nil {
			rfield.SetFloat(fvalue)
		}

	case reflect.Float64:
		fvalue, err := strconv.ParseFloat(value, 64)
		if err == nil {
			rfield.SetFloat(fvalue)
		}

	case reflect.Bool:
		if value == "true" {
			rfield.SetBool(true)
		} else {
			rfield.SetBool(false)
		}

	case reflect.String:
		if value != "" {
			rfield.SetString(value)
		}

	default:
		fmt.Println("SetModelField(): unhandled field type:", rfield.Kind())
	}

}

func (crud basicCRUD) ParseSubEntity(modelPtr interface{}, form url.Values, prefix string, idx int) int {
	rval := reflect.ValueOf(modelPtr).Elem()
	typ := rval.Type()
	count := 0

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)

		if p.Name == "Id" {
			continue
		}

		fieldName := prefix + "[" + strconv.Itoa(idx) + "][" + p.Name + "]"
		value := form.Get(fieldName)
		if value != "" {
			count = count + 1
		}

		SetModelField(rval, i, value)
	}

	return count
}

func (crud basicCRUD) FromForm(modelPtr interface{}, form url.Values) {
	rval := reflect.ValueOf(modelPtr).Elem()
	typ := rval.Type()

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)

		if p.Name == "Id" {
			continue
		}

		SetModelField(rval, i, form.Get(p.Name))
	}
}

func (crud basicCRUD) UploadFiles(modelPtr interface{}, req *http.Request, uploader FileUploader) {
	if req.MultipartForm == nil {
		return
	}

	fieldMap := make(map[string]reflect.Value)
	rval := reflect.ValueOf(modelPtr).Elem()
	typ := rval.Type()

	// create field map
	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)

		if p.Name == "Id" {
			continue
		}

		fieldMap[p.Name] = rval.Field(i)
	}

	// process files
	for key, _ := range req.MultipartForm.File {
		fileField, fileHeader, err := req.FormFile(key)
		if err != nil {
			fmt.Println(err)
			return
		}

		defer fileField.Close()

		_ = fileHeader

		rootDir, generatedName := uploader(key, fileField, fileHeader)

		// remove old file
		oldFile := fieldMap[key].String()
		oldFile = rootDir + oldFile
		if _, err := os.Stat(oldFile); err == nil {
			os.Remove(oldFile)
			fmt.Println("removed old image:", oldFile)
		}

		// persist uploaded path in model
		fieldMap[key].SetString(generatedName)

		fmt.Println("uploading to:", rootDir+generatedName)

		out, err := os.Create(rootDir + generatedName)
		if err != nil {
			fmt.Printf("Unable to create the file for writing. Check your write access privilege")
			return
		}

		defer out.Close()

		// write the content from POST to the file
		_, err = io.Copy(out, fileField)
		if err != nil {
			fmt.Println(err)
		}
	}
}
