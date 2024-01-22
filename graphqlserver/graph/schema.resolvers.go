package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.42

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/intelops/kubviz/graphqlserver/graph/model"
)

// AllNamespaceData is the resolver for the allNamespaceData field.
func (r *queryResolver) AllNamespaceData(ctx context.Context) ([]*model.NamespaceData, error) {
	var namespaceDataList []*model.NamespaceData

	namespaces, err := r.fetchNamespacesFromDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching namespaces: %v", err)
	}

	for _, ns := range namespaces {
		outdatedImages, err := r.fetchOutdatedImages(ctx, ns)
		if err != nil {
			fmt.Printf("error fetching outdated images for namespace %s: %v\n", ns, err)
			outdatedImages = []*model.OutdatedImage{}
		}

		kubeScores, err := r.fetchKubeScores(ctx, ns)
		if err != nil {
			fmt.Printf("error fetching kube scores for namespace %s: %v\n", ns, err)
			kubeScores = []*model.KubeScore{}
		}

		resources, err := r.fetchResources(ctx, ns)
		if err != nil {
			fmt.Printf("error fetching resources for namespace %s: %v\n", ns, err)
			resources = []*model.Resource{}
		}

		nd := &model.NamespaceData{
			Namespace:      ns,
			OutdatedImages: outdatedImages,
			KubeScores:     kubeScores,
			Resources:      resources,
		}

		namespaceDataList = append(namespaceDataList, nd)
	}

	return namespaceDataList, nil
}

// AllEvents is the resolver for the allEvents field.
func (r *queryResolver) AllEvents(ctx context.Context) ([]*model.Event, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT ClusterName, Id, EventTime, OpType, Name, Namespace, Kind, Message, Reason, Host, Event, FirstTime, LastTime, ExpiryDate FROM events`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.Event{}, nil
		}
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		var e model.Event
		if err := rows.Scan(&e.ClusterName, &e.ID, &e.EventTime, &e.OpType, &e.Name, &e.Namespace, &e.Kind, &e.Message, &e.Reason, &e.Host, &e.Event, &e.FirstTime, &e.LastTime, &e.ExpiryDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return events, nil
}

// AllRakkess is the resolver for the allRakkess field.
func (r *queryResolver) AllRakkess(ctx context.Context) ([]*model.Rakkess, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT ClusterName, Name, Create, Delete, List, Update, EventTime, ExpiryDate FROM rakkess`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.Rakkess{}, nil
		}
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var rakkessRecords []*model.Rakkess
	for rows.Next() {
		var r model.Rakkess
		if err := rows.Scan(&r.ClusterName, &r.Name, &r.Create, &r.Delete, &r.List, &r.Update, &r.EventTime, &r.ExpiryDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		rakkessRecords = append(rakkessRecords, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return rakkessRecords, nil
}

// AllDeprecatedAPIs is the resolver for the allDeprecatedAPIs field.
func (r *queryResolver) AllDeprecatedAPIs(ctx context.Context) ([]*model.DeprecatedAPI, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT ClusterName, ObjectName, Description, Kind, Deprecated, Scope, EventTime, ExpiryDate FROM DeprecatedAPIs`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.DeprecatedAPI{}, nil
		}
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var deprecatedAPIs []*model.DeprecatedAPI
	for rows.Next() {
		var d model.DeprecatedAPI
		var deprecatedInt uint8
		if err := rows.Scan(&d.ClusterName, &d.ObjectName, &d.Description, &d.Kind, &deprecatedInt, &d.Scope, &d.EventTime, &d.ExpiryDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		// Convert uint8 to bool
		deprecatedBool := deprecatedInt != 0
		d.Deprecated = &deprecatedBool

		deprecatedAPIs = append(deprecatedAPIs, &d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return deprecatedAPIs, nil
}

// AllDeletedAPIs is the resolver for the allDeletedAPIs field.
func (r *queryResolver) AllDeletedAPIs(ctx context.Context) ([]*model.DeletedAPI, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT ClusterName, ObjectName, Group, Kind, Version, Name, Deleted, Scope, EventTime, ExpiryDate FROM DeletedAPIs`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.DeletedAPI{}, nil
		}
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var deletedAPIs []*model.DeletedAPI
	for rows.Next() {
		var d model.DeletedAPI
		var deletedInt uint8
		if err := rows.Scan(&d.ClusterName, &d.ObjectName, &d.Group, &d.Kind, &d.Version, &d.Name, &deletedInt, &d.Scope, &d.EventTime, &d.ExpiryDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		// Convert uint8 to bool
		deletedBool := deletedInt != 0
		d.Deleted = &deletedBool

		deletedAPIs = append(deletedAPIs, &d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return deletedAPIs, nil
}

// AllGetAllResources is the resolver for the allGetAllResources field.
func (r *queryResolver) AllGetAllResources(ctx context.Context) ([]*model.GetAllResource, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT ClusterName, Namespace, Kind, Resource, Age, EventTime, ExpiryDate FROM getall_resources`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.GetAllResource{}, nil
		}
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var resources []*model.GetAllResource
	for rows.Next() {
		var res model.GetAllResource
		if err := rows.Scan(&res.ClusterName, &res.Namespace, &res.Kind, &res.Resource, &res.Age, &res.EventTime, &res.ExpiryDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		resources = append(resources, &res)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return resources, nil
}

// AllTrivySBOMs is the resolver for the allTrivySBOMs field.
func (r *queryResolver) AllTrivySBOMs(ctx context.Context) ([]*model.TrivySbom, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT id, cluster_name, image_name, package_name, package_url, bom_ref, serial_number, version, bom_format, ExpiryDate FROM trivysbom`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.TrivySbom{}, nil
		}
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var sboms []*model.TrivySbom
	for rows.Next() {
		var s model.TrivySbom
		if err := rows.Scan(&s.ID, &s.ClusterName, &s.ImageName, &s.PackageName, &s.PackageURL, &s.BomRef, &s.SerialNumber, &s.Version, &s.BomFormat, &s.ExpiryDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		sboms = append(sboms, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return sboms, nil
}

// AllTrivyImages is the resolver for the allTrivyImages field.
func (r *queryResolver) AllTrivyImages(ctx context.Context) ([]*model.TrivyImage, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT id, cluster_name, artifact_name, vul_id, vul_pkg_id, vul_pkg_name, vul_installed_version, vul_fixed_version, vul_title, vul_severity, vul_published_date, vul_last_modified_date, ExpiryDate FROM trivyimage`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.TrivyImage{}, nil
		}
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var images []*model.TrivyImage
	for rows.Next() {
		var img model.TrivyImage
		if err := rows.Scan(&img.ID, &img.ClusterName, &img.ArtifactName, &img.VulID, &img.VulPkgID, &img.VulPkgName, &img.VulInstalledVersion, &img.VulFixedVersion, &img.VulTitle, &img.VulSeverity, &img.VulPublishedDate, &img.VulLastModifiedDate, &img.ExpiryDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		images = append(images, &img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return images, nil
}

// AllKubeScores is the resolver for the allKubeScores field.
func (r *queryResolver) AllKubeScores(ctx context.Context) ([]*model.Kubescore, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT id, clustername, object_name, kind, apiVersion, name, namespace, target_type, description, path, summary, file_name, file_row, EventTime, ExpiryDate FROM kubescore`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.Kubescore{}, nil
		}
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var kubeScores []*model.Kubescore
	for rows.Next() {
		var ks model.Kubescore
		if err := rows.Scan(&ks.ID, &ks.ClusterName, &ks.ObjectName, &ks.Kind, &ks.APIVersion, &ks.Name, &ks.Namespace, &ks.TargetType, &ks.Description, &ks.Path, &ks.Summary, &ks.FileName, &ks.FileRow, &ks.EventTime, &ks.ExpiryDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		kubeScores = append(kubeScores, &ks)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return kubeScores, nil
}

// AllTrivyVuls is the resolver for the allTrivyVuls field.
func (r *queryResolver) AllTrivyVuls(ctx context.Context) ([]*model.TrivyVul, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT id, cluster_name, namespace, kind, name, vul_id, vul_vendor_ids, vul_pkg_id, vul_pkg_name, vul_pkg_path, vul_installed_version, vul_fixed_version, vul_title, vul_severity, vul_published_date, vul_last_modified_date, ExpiryDate FROM trivy_vul`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.TrivyVul{}, nil
		}
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var trivyVuls []*model.TrivyVul
	for rows.Next() {
		var tv model.TrivyVul
		if err := rows.Scan(&tv.ID, &tv.ClusterName, &tv.Namespace, &tv.Kind, &tv.Name, &tv.VulID, &tv.VulVendorIds, &tv.VulPkgID, &tv.VulPkgName, &tv.VulPkgPath, &tv.VulInstalledVersion, &tv.VulFixedVersion, &tv.VulTitle, &tv.VulSeverity, &tv.VulPublishedDate, &tv.VulLastModifiedDate, &tv.ExpiryDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		trivyVuls = append(trivyVuls, &tv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return trivyVuls, nil
}

// AllTrivyMisconfigs is the resolver for the allTrivyMisconfigs field.
func (r *queryResolver) AllTrivyMisconfigs(ctx context.Context) ([]*model.TrivyMisconfig, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT id, cluster_name, namespace, kind, name, misconfig_id, misconfig_avdid, misconfig_type, misconfig_title, misconfig_desc, misconfig_msg, misconfig_query, misconfig_resolution, misconfig_severity, misconfig_status, EventTime, ExpiryDate FROM trivy_misconfig`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.TrivyMisconfig{}, nil
		}
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var misconfigs []*model.TrivyMisconfig
	for rows.Next() {
		var tm model.TrivyMisconfig
		if err := rows.Scan(&tm.ID, &tm.ClusterName, &tm.Namespace, &tm.Kind, &tm.Name, &tm.MisconfigID, &tm.MisconfigAvdid, &tm.MisconfigType, &tm.MisconfigTitle, &tm.MisconfigDesc, &tm.MisconfigMsg, &tm.MisconfigQuery, &tm.MisconfigResolution, &tm.MisconfigSeverity, &tm.MisconfigStatus, &tm.EventTime, &tm.ExpiryDate); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}
		misconfigs = append(misconfigs, &tm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return misconfigs, nil
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
