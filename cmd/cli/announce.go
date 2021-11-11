package main

import (
	"context"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newAnnounceCmd() *cobra.Command {
	var leaderClient *ServerClient

	// Where the new settings will be put
	announcement := &models.Announcement{}
	var announcementType string
	var announcementAnimation string

	cmd := &cobra.Command{
		Use:   "announce [string to announce]",
		Short: `Announce a message across the ticker wall.`,
		Long:  `Announce a message across the ticker wall.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Create a leader client...
			leader, _ := cmd.Flags().GetString("leader")
			leaderClient, err = NewServerClient(leader)
			if err != nil {
				return err
			}
			logrus.Debug("Connected to Leader.")

			announcement.Animation = int32(getAnnouncementAnimation(announcementAnimation))
			announcement.AnnouncementType = int32(getAnnouncementType(announcementType))
			announcement.Message = args[0]

			if _, err = leaderClient.client.Announce(context.Background(), announcement); err != nil {
				return err
			}

			logrus.Info("Announcement Sent.")

			return nil
		},
	}

	// Announcement params.
	cmd.Flags().StringVarP(&announcementType, "type", "t", "info", "Announcement type. This determines the colors of the announcement. Valid options: ( info, danger, success )")
	cmd.Flags().StringVarP(&announcementAnimation, "animation", "n", "elastic", "Announcement animation. Valid options: ( elastic, ease, back, bounce )")
	cmd.Flags().Int64VarP(&announcement.LifespanMS, "lifespan", "i", 2000, "How long the message will be displayed on the ticker wall, in milliseconds.")

	// Dont auto sort flags.
	cmd.Flags().SortFlags = false

	return cmd
}

func getAnnouncementType(flagString string) models.AnnouncementType {
	switch flagString {
	case "info":
		return models.AnnouncementTypeInfo
	case "danger":
		return models.AnnouncementTypeDanger
	case "success":
		return models.AnnouncementTypeSuccess
	default:
		return models.AnnouncementTypeInfo
	}
}

func getAnnouncementAnimation(flagString string) models.AnnouncementAnimation {
	switch flagString {
	case "elastic":
		return models.AnnouncementAnimationElastic
	case "bounce":
		return models.AnnouncementAnimationBounce
	case "ease":
		return models.AnnouncementAnimationEase
	case "back":
		return models.AnnouncementAnimationBack
	default:
		return models.AnnouncementAnimationElastic
	}
}
