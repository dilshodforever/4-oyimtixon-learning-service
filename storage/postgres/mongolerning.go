package postgres

import (
	"context"
	"errors"
	"log"
	"log/slog"

	"github.com/dilshodforever/4-oyimtixon-game-service/genprotos/game"
	pb "github.com/dilshodforever/4-oyimtixon-game-service/genprotos/learning"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type LearningStorage struct {
	db *mongo.Database
}

func NewLearningStorage(db *mongo.Database) *LearningStorage {
	return &LearningStorage{db: db}
}

func (ls *LearningStorage) GetTopics(req *pb.GetTopicsRequest) (*pb.GetTopicsResponse, error) {
	coll := ls.db.Collection("topics")
	cursor, err := coll.Find(context.Background(), bson.D{})
	if err != nil {
		log.Printf("Failed to get topics: %v", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	var topics []*pb.Topic
	for cursor.Next(context.Background()) {
		var topic pb.Topic
		if err := cursor.Decode(&topic); err != nil {
			log.Printf("Failed to decode topic: %v", err)
			return nil, err
		}
		topics = append(topics, &topic)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Cursor error: %v", err)
		return nil, err
	}

	return &pb.GetTopicsResponse{Topics: topics}, nil
}

func (ls *LearningStorage) GetTopic(req *pb.GetTopicRequest) (*pb.Topic, error) {
	coll := ls.db.Collection("topics")
	filter := bson.D{{Key: "id", Value: req.TopicId}}
	var topic pb.Topic
	err := coll.FindOne(context.Background(), filter).Decode(&topic)
	if err != nil {
		log.Printf("Failed to get topic: %v", err)
		return nil, err
	}
	return &topic, nil
}

func (ls *LearningStorage) CompleteTopic(req *pb.CompleteTopicRequest) (*pb.CompleteTopicResponse, error) {
	coll := ls.db.Collection("topics")
	filter := bson.D{{Key: "id", Value: req.TopicId}}
	var topic pb.Topic
	err := coll.FindOne(context.Background(), filter).Decode(&topic)
	if err != nil {
		log.Printf("Failed to get topic: %v", err)
		return nil, err
	}
	res, err:=ls.UpdateUserXp(&pb.Update{UserId: req.Userid, Xps: 50})
	if err != nil {
		log.Printf("Failed to update userxps: %v", err)
		return nil, err
	}
	ls.UpdateComplateds(&pb.CalculateCompleteds{TopicsCompleted: 1})
	return &pb.CompleteTopicResponse{
		Message:  "Topic completed successfully",
		XpEarned: res.XpEarned,
	}, nil
}

func (ls *LearningStorage) GetQuiz(req *pb.GetQuizRequest) (*pb.Quiz, error) {
	coll := ls.db.Collection("topics")
	filter := bson.D{{Key: "quiz.id", Value: req.QuizId}}
	var quiz pb.Quiz
	err := coll.FindOne(context.Background(), filter).Decode(&quiz)
	if err != nil {
		log.Printf("Failed to get quiz: %v", err)
		return nil, err
	}
	return &quiz, nil
}

func (ls *LearningStorage) SubmitQuiz(req *pb.SubmitQuizRequest) (*pb.SubmitQuizResponse, error) {
	coll := ls.db.Collection("topics")
	filter := bson.D{{Key: "topics.quiz.id", Value: req.QuizId}}

	var level pb.Topic
	err := coll.FindOne(context.Background(), filter).Decode(&level)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("No level found with id: %v", req.QuizId)
			return nil, errors.New("no rows in result set")
		}
		log.Printf("Failed to decode level: %v", err)
		return nil, err
	}

	var challenge *pb.Quiz
	for _, ch := range level.Quiz {
		if ch.Id == req.QuizId {
			challenge = ch
			break
		}
	}
	if challenge == nil {
		log.Printf("Challenge with id: %v not found in level: %v", req.QuizId, level.Id)
		return nil, errors.New("challenge with id not found in level")
	}

	var submitsresult pb.SubmitQuizResponse
	submitsresult.TotalQuestions = int32(len(challenge.Questions))

	for i := 0; i < len(req.Answers); i++ {

		for j := 0; j < len(challenge.Questions); j++ {

			if req.Answers[i].SelectedOption == challenge.Questions[i].CorrectOption && req.Answers[i].QuestionId == challenge.Questions[j].Id {
				submitsresult.XpEarned += 10
				submitsresult.TotalQuestions++
				submitsresult.CorrectAnswers[i].QuestionId = challenge.Questions[j].Id
				submitsresult.CorrectAnswers[i].SelectedOption = req.Answers[i].SelectedOption
			}
		}
	}
	if submitsresult.XpEarned == 0 {
		submitsresult.Feedback = "Keep practicing! You can improve"
		return &submitsresult, nil
	}
	switch submitsresult.TotalQuestions {
	case int32(len(req.Answers)):
		submitsresult.Feedback = "Excellent! You have a good understanding of quantum superposition."
	case int32(len(req.Answers)) / 2:
		submitsresult.Feedback = "Nice! You're on the right track."
	default:
		submitsresult.Feedback = "Keep practicing! You can improve."
	}
	_, err = ls.UpdateUserXp(&pb.Update{UserId: req.Userid, Xps: submitsresult.XpEarned})
	if err != nil {
		slog.Info(err.Error())
		return nil, err
	}
	_,err=ls.UpdateComplateds(&pb.CalculateCompleteds{QuizzesCompleted:int32(len(req.Answers))})
	if err != nil {
		slog.Info(err.Error())
		return nil, err
	}
	return &submitsresult, nil
}

func (ls *LearningStorage) GetResources(req *pb.GetResourcesRequest) (*pb.GetResourcesResponse, error) {
	coll := ls.db.Collection("resources")
	cursor, err := coll.Find(context.Background(), bson.D{})
	if err != nil {
		log.Printf("Failed to get resources: %v", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	var resources []*pb.Resource
	for cursor.Next(context.Background()) {
		var resource pb.Resource
		if err := cursor.Decode(&resource); err != nil {
			log.Printf("Failed to decode resource: %v", err)
			return nil, err
		}
		resources = append(resources, &resource)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Cursor error: %v", err)
		return nil, err
	}

	return &pb.GetResourcesResponse{Resources: resources}, nil
}

func (ls *LearningStorage) CompleteResource(req *pb.CompleteResourceRequest) (*pb.CompleteResourceResponse, error) {
	res, err:=ls.UpdateUserXp(&pb.Update{UserId: req.UserId, Xps: 10})
	if err != nil {
		log.Printf("Failed to update userxps: %v", err)
		return nil, err
	}
	_,err=ls.UpdateComplateds(&pb.CalculateCompleteds{ResourcesCompleted: 1})
	if err != nil {
		log.Printf("Failed to update complates: %v", err)
		return nil, err
	}
	return &pb.CompleteResourceResponse{
		Message:  "Resource completed successfully",
		XpEarned: res.XpEarned,
	}, nil
}

func (ls *LearningStorage) GetProgress(req *pb.GetProgressRequest) (*pb.ProgressResponse, error) {
	coll := ls.db.Collection("Complateds")
	filter := bson.D{{Key: "user_id", Value: req.Userid}}

	var completedData pb.CalculateCompleteds
	err := coll.FindOne(context.Background(), filter).Decode(&completedData)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("No progress found for user_id: %v", req.Userid)
			return nil, errors.New("no progress data found")
		}
		log.Printf("Failed to get progress: %v", err)
		return nil, err
	}

	res, err:=ls.GetTopics(&pb.GetTopicsRequest{})
	if err != nil {
		log.Printf("Failed to get topics: %v", err)
		return nil, err
	}
	rest, err:=ls.GetResources(&pb.GetResourcesRequest{})
	if err != nil {
		log.Printf("Failed to get topicsquestions: %v", err)
		return nil, err
	}
	totalTopics := len(res.Topics)
	totalQuizzes := ls.CountQuestions(res.Topics)
	totalResources := len(rest.Resources)

	overallProgress := float32(completedData.TopicsCompleted+completedData.QuizzesCompleted+completedData.ResourcesCompleted) /
		float32(totalTopics+totalQuizzes+totalResources) * 100

	progress := &pb.ProgressResponse{
		TopicsCompleted:    completedData.TopicsCompleted,
		TotalTopics:        int32(totalTopics),
		QuizzesCompleted:   completedData.QuizzesCompleted,
		TotalQuizzes:       int32(totalQuizzes),
		ResourcesCompleted: completedData.ResourcesCompleted,
		TotalResources:     int32(totalResources),
		OverallProgress:    overallProgress,
	}

	return progress, nil
}

func (ls *LearningStorage) GetRecommendations(req *pb.GetRecommendationsRequest) (*pb.GetRecommendationsResponse, error) {
	coll := ls.db.Collection("recommendations")
	cursor, err := coll.Find(context.Background(), bson.D{})
	if err != nil {
		log.Printf("Failed to get topics: %v", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	
	recommendations := pb.GetRecommendationsResponse{} 
	for cursor.Next(context.Background()) {
		var topic pb.Topics
		if err := cursor.Decode(&topic); err != nil {
			log.Printf("Failed to decode topic: %v", err)
			return nil, err
		}
		recommendations.Recommendations = append(recommendations.Recommendations, &topic)
	}
	return &recommendations, nil
}

func (ls *LearningStorage) SubmitFeedback(req *pb.SubmitFeedbackRequest) (*pb.SubmitFeedbackResponse, error) {
	coll := ls.db.Collection("feedbacks")
	documents := []interface{}{
		bson.D{
			{Key: "Userid", Value: req.Userid},
			{Key: "TopicId", Value: req.TopicId},
			{Key: "rating", Value: req.Rating},
			{Key: "comment", Value: req.Comment},
		},
	}

	_, err := coll.InsertMany(context.Background(), documents)
	if err != nil {
		log.Printf("Failed to insert feedback: %v", err)
		return nil, err
	}

	res, err:=ls.UpdateUserXp(&pb.Update{UserId:req.Userid, Xps: 10})
	if err != nil {
		log.Printf("Failed to update xps: %v", err)
		return nil, err
	}
	return &pb.SubmitFeedbackResponse{
		Message:  "Feedback submitted successfully",
		XpEarned: res.XpEarned,
	}, nil
}

	

func (ls *LearningStorage) GetChallenges(req *pb.GetChallengesRequest) (*pb.GetChallengesResponse, error) {
	coll := ls.db.Collection("challenges")
	cursor, err := coll.Find(context.Background(), bson.D{})
	if err != nil {
		log.Printf("Failed to get challenges: %v", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	var challenges []*pb.Challenge
	for cursor.Next(context.Background()) {
		var challenge pb.Challenge
		if err := cursor.Decode(&challenge); err != nil {
			log.Printf("Failed to decode challenge: %v", err)
			return nil, err
		}
		challenges = append(challenges, &challenge)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Cursor error: %v", err)
		return nil, err
	}

	return &pb.GetChallengesResponse{Challenges: challenges}, nil
}

func (g *LearningStorage) UpdateUserXp(req *pb.Update) (*game.CompleteLevelResponse, error) {
	coll := g.db.Collection("user_levels")
	filter := bson.D{
		{Key: "user_id", Value: req.UserId},
	}
	var user game.Level
	err := coll.FindOne(context.Background(), filter).Decode(&user)
	if err != nil {
		log.Printf("Failed to get challenge: %v", err)
		return nil, err
	}
	xps := req.Xps + user.RequiredXp
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "user_xp", Value: xps},
		}},
	}
	_, err = coll.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("Failed to complete level: %v", err)
		return nil, err
	}

	return &game.CompleteLevelResponse{XpEarned: xps}, nil
}




func (g *LearningStorage) UpdateComplateds(req *pb.CalculateCompleteds) (*game.CompleteLevelResponse, error) {
	coll := g.db.Collection("Complateds")
	filter := bson.D{
		{Key: "user_id", Value: req.Userid},
	}
	
	var gets pb.CalculateCompleteds
	err := coll.FindOne(context.Background(), filter).Decode(&gets)
	if err != nil {
		log.Printf("Failed to get challenge: %v", err)
		return nil, err
	}

	update := bson.D{
		{Key: "$inc", Value: bson.D{}},
	}

	if req.QuizzesCompleted > 0 {
		update[0].Value = append(update[0].Value.(bson.D), bson.E{Key: "QuizzesCompleted", Value: req.QuizzesCompleted})
	}
	if req.ResourcesCompleted > 0 {
		update[0].Value = append(update[0].Value.(bson.D), bson.E{Key: "ResourcesCompleted", Value: req.ResourcesCompleted})
	}
	if req.TopicsCompleted > 0 {
		update[0].Value = append(update[0].Value.(bson.D), bson.E{Key: "TopicsCompleted", Value: req.TopicsCompleted})
	}

	if len(update[0].Value.(bson.D)) == 0 {
		return &game.CompleteLevelResponse{Message: "Nothing to update"}, nil
	}

	_, err = coll.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("Failed to complete level: %v", err)
		return nil, err
	}

	return &game.CompleteLevelResponse{Message: "Success!!!"}, nil
}




func (ls *LearningStorage) CountQuestions(req []*pb.Topic) int {
	var count int
	for i := 0; i < len(req); i++ {
		count+=len(req[i].Quiz)
	}
	return count
}



func (g *LearningStorage) StartGame(req *pb.Update) (*game.CompleteLevelResponse, error) {
	coll := g.db.Collection("user_levels")
	
	// Construct the document to insert
	document := bson.D{
		{Key: "user_id", Value: req.UserId},
		{Key: "user_xp", Value: 0},
	}

	_, err := coll.InsertOne(context.Background(), document)
	if err != nil {
		log.Printf("Failed to insert user level: %v", err)
		return nil, err
	}
	
	return &game.CompleteLevelResponse{Message: "Success"}, nil
}
