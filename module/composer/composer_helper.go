package composer

import (
	"context"
	"github.com/murphysecurity/murphysec/errors"
	"github.com/murphysecurity/murphysec/utils"
	"os/exec"
)

func doComposerInstall(ctx context.Context, projectDir string) error {
	composerPath, e := exec.LookPath("composer")
	if e != nil {
		return errors.WithCause(ErrNoComposerFound, e)
	}
	logger := utils.UseLogger(ctx)
	c := exec.CommandContext(ctx, composerPath, "install", "--ignore-platform-reqs", "--no-progress", "--no-dev", "--no-autoloader", "--no-scripts", "--no-interaction")
	c.Dir = projectDir
	logger.Sugar().Infof("Command: %s", c.String())
	defer logger.Info("doComposerInstall terminated")
	lp := utils.NewLogPipe(logger, "composer")
	defer lp.Close()
	c.Stderr = lp
	c.Stdout = lp
	if e := c.Run(); e != nil {
		return errors.Wrap(e, "composer install command execute failed")
	}
	return nil
}
