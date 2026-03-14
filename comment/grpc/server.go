package grpc

import (
	"context"
	"learning_go/webook/api/proto/gen/comment/comv1"
	"learning_go/webook/comment/domain"
	"learning_go/webook/comment/service"
	"math"
)

type CommentServiceServer struct {
	comv1.UnimplementedCommentServiceServer
	svc service.CommentService
}

func NewCommentServiceServer(svc service.CommentService) *CommentServiceServer {
	return &CommentServiceServer{svc: svc}
}

func (c *CommentServiceServer) GetCommentList(ctx context.Context, req *comv1.GetCommentListReq) (*comv1.GetCommentListResp, error) {
	minId := req.GetMinId()
	if minId <= 0 {
		minId = math.MaxInt
	}
	domainComments, err := c.svc.GetCommentList(ctx, req.GetBiz(), req.GetBizId(), minId, req.GetLimit())
	if err != nil {
		return nil, err
	}
	return &comv1.GetCommentListResp{Comments: c.toDTO(domainComments)}, err
}

func (c *CommentServiceServer) GetMoreReplies(ctx context.Context, req *comv1.GetMoreRepliesReq) (*comv1.GetMoreRepliesResp, error) {
	maxId := req.GetMaxId()
	if maxId <= 0 {
		maxId = 0
	}
	domainComments, err := c.svc.GetMoreReplies(ctx, req.GetRid(), req.GetMaxId(), req.GetLimit())
	if err != nil {
		return nil, err
	}
	return &comv1.GetMoreRepliesResp{Comments: c.toDTO(domainComments)}, err
}

func (c *CommentServiceServer) CreateComment(ctx context.Context, req *comv1.CreateCommentReq) (*comv1.CreateCommentResp, error) {
	err := c.svc.CreateComment(ctx, c.toDomain(req.GetComment()))
	return &comv1.CreateCommentResp{}, err
}

func (c *CommentServiceServer) DeleteComment(ctx context.Context, req *comv1.DeleteCommentReq) (*comv1.DeleteCommentResp, error) {
	err := c.svc.DeleteComment(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &comv1.DeleteCommentResp{}, err
}

func (c *CommentServiceServer) toDomain(comment *comv1.Comment) domain.Comment {
	domainComment := domain.Comment{
		Id:      comment.Id,
		Uid:     comment.Uid,
		Biz:     comment.Biz,
		BizId:   comment.BizId,
		Content: comment.Content,
		RootId:  comment.RootId,
		PID:     comment.PID,
		CTime:   comment.CTime,
		UTime:   comment.UTime,
	}
	if comment.ParentComment != nil {
		domainComment.ParentComment = &domain.Comment{Id: comment.ParentComment.Id}
	}
	return domainComment
}

func (c *CommentServiceServer) toDTO(comments []domain.Comment) []*comv1.Comment {
	rpcComments := make([]*comv1.Comment, 0, len(comments))
	for _, domainComment := range comments {
		rpcComment := &comv1.Comment{
			Id:      domainComment.Id,
			Uid:     domainComment.Uid,
			Biz:     domainComment.Biz,
			BizId:   domainComment.BizId,
			Content: domainComment.Content,
			RootId:  domainComment.RootId,
			CTime:   domainComment.CTime,
			UTime:   domainComment.UTime,
		}
		if domainComment.ParentComment != nil {
			rpcComment.ParentComment = &comv1.Comment{
				Id: domainComment.ParentComment.Id,
			}
		}
		rpcComments = append(rpcComments, rpcComment)
	}
	rpcCommentMap := make(map[int64]*comv1.Comment, len(rpcComments))
	for _, rpcComment := range rpcComments {
		rpcCommentMap[rpcComment.Id] = rpcComment
	}
	for _, domainComment := range comments {
		rpcComment := rpcCommentMap[domainComment.Id]
		if domainComment.ParentComment != nil {
			val, ok := rpcCommentMap[domainComment.ParentComment.Id]
			if ok {
				rpcComment.ParentComment = val
			}
		}
	}
	return rpcComments
}

func (c *CommentServiceServer) mustEmbedUnimplementedCommentServiceServer() {
	panic("implement me")
}
