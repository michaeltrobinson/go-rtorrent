package rtorrent

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestRTorrent(t *testing.T) {
	/*
		These tests rely on a local instance of rtorrent to be running in a clean state.
		Use the included `test.sh` script to run these tests.
	*/
	client := New("http://localhost/RPC2", false)

	t.Run("get ip", func(t *testing.T) {
		_, err := client.IP()
		require.NoError(t, err)
		// Don't assert anything about the response, differs based upon the environment
	})

	t.Run("get name", func(t *testing.T) {
		name, err := client.Name()
		require.NoError(t, err)
		require.NotEmpty(t, name)
	})

	t.Run("down total", func(t *testing.T) {
		total, err := client.DownTotal()
		require.NoError(t, err)
		require.Zero(t, total, "expected no data to be transferred yet")
	})

	t.Run("up total", func(t *testing.T) {
		total, err := client.UpTotal()
		require.NoError(t, err)
		require.Zero(t, total, "expected no data to be transferred yet")
	})

	t.Run("down rate", func(t *testing.T) {
		rate, err := client.DownRate()
		require.NoError(t, err)
		require.Zero(t, rate, "expected no download yet")
	})

	t.Run("up rate", func(t *testing.T) {
		rate, err := client.UpRate()
		require.NoError(t, err)
		require.Zero(t, rate, "expected no upload yet")
	})

	t.Run("get no torrents", func(t *testing.T) {
		torrents, err := client.GetTorrents(ViewMain)
		require.NoError(t, err)
		require.Empty(t, torrents, "expected no torrents to be added yet")
	})

	t.Run("add", func(t *testing.T) {
		t.Run("by url", func(t *testing.T) {
			err := client.Add("https://torrent.ubuntu.com/releases/focal/release/live-server/ubuntu-20.04.2-live-server-arm64.iso.torrent")
			require.NoError(t, err)

			t.Run("get torrent", func(t *testing.T) {
				// It will take some time to appear, so retry a few times
				tries := 0
				var torrents []Torrent
				var err error
				for {
					<-time.After(time.Second)
					torrents, err = client.GetTorrents(ViewMain)
					require.NoError(t, err)
					if len(torrents) > 0 {
						break
					}
					if tries > 10 {
						require.NoError(t, errors.Errorf("torrent did not show up in time"))
					}
					tries++
				}
				require.NotEmpty(t, torrents)
				require.Len(t, torrents, 1)
				require.Equal(t, "B185765BE59BC56AB29A4860A5820863BF0512AD", torrents[0].Hash)
				require.Equal(t, "ubuntu-20.04.2-live-server-arm64.iso", torrents[0].Name)
				require.Equal(t, "", torrents[0].Label)
				require.Equal(t, 1193754624, torrents[0].Size)
				require.Equal(t, "/downloads/incoming/ubuntu-20.04.2-live-server-arm64.iso", torrents[0].Path)
				require.False(t, torrents[0].Completed)

				t.Run("get files", func(t *testing.T) {
					files, err := client.GetFiles(torrents[0])
					require.NoError(t, err)
					require.NotEmpty(t, files)
					require.Len(t, files, 1)
					for _, f := range files {
						require.NotEmpty(t, f.Path)
						require.NotZero(t, f.Size)
					}
				})

				t.Run("single get", func(t *testing.T) {
					torrent, err := client.GetTorrent(torrents[0].Hash)
					require.NoError(t, err)
					require.NotEmpty(t, torrent.Hash)
					require.NotEmpty(t, torrent.Name)
					require.NotEmpty(t, torrent.Path)
					require.NotEmpty(t, torrent.Size)
				})

				t.Run("change label", func(t *testing.T) {
					err := client.SetLabel(torrents[0], "TestLabel")
					require.NoError(t, err)

					// It will take some time to change, so try a few times
					tries := 0
					for {
						<-time.After(time.Second)
						torrents, err = client.GetTorrents(ViewMain)
						require.NoError(t, err)
						require.Len(t, torrents, 1)
						if torrents[0].Label != "" {
							break
						}
						if tries > 10 {
							require.NoError(t, errors.Errorf("torrent label did not change in time"))
						}
						tries++
					}
					require.Equal(t, "TestLabel", torrents[0].Label)
				})

				t.Run("get status", func(t *testing.T) {
					var status Status
					var err error
					// It may take some time for the download to start
					tries := 0
					for {
						<-time.After(time.Second)
						status, err = client.GetStatus(torrents[0])
						require.NoError(t, err)
						t.Logf("Status = %+v", status)
						if status.CompletedBytes > 0 {
							break
						}
						if tries > 10 {
							require.NoError(t, errors.Errorf("torrent did not start in time"))
						}
						tries++
					}

					require.False(t, status.Completed)
					require.NotZero(t, status.CompletedBytes)
					require.NotZero(t, status.DownRate)
					require.NotZero(t, status.Size)
					// require.NotZero(t, status.UpRate)
					//require.NotZero(t, status.Ratio)
				})

				t.Run("delete torrent", func(t *testing.T) {
					err := client.Delete(torrents[0])
					require.NoError(t, err)

					torrents, err := client.GetTorrents(ViewMain)
					require.NoError(t, err)
					require.Empty(t, torrents)

					t.Run("get torrent", func(t *testing.T) {
						// It will take some time to disappear, so retry a few times
						tries := 0
						var torrents []Torrent
						var err error
						for {
							<-time.After(time.Second)
							torrents, err = client.GetTorrents(ViewMain)
							require.NoError(t, err)
							if len(torrents) == 0 {
								break
							}
							if tries > 10 {
								require.NoError(t, errors.Errorf("torrent did not delete in time"))
							}
							tries++
						}
						require.Empty(t, torrents)
					})
				})

			})
		})

		t.Run("by url (stopped)", func(t *testing.T) {
			label := DLabel.SetValue("test-label")
			err := client.AddStopped("https://torrent.ubuntu.com/releases/focal/release/live-server/ubuntu-20.04.2-live-server-arm64.iso.torrent", label)
			require.NoError(t, err)

			t.Run("get torrent", func(t *testing.T) {
				// It will take some time to appear, so retry a few times
				tries := 0
				var torrents []Torrent
				var err error
				for {
					<-time.After(time.Second)
					torrents, err = client.GetTorrents(ViewStopped)
					require.NoError(t, err)
					if len(torrents) > 0 {
						break
					}
					if tries > 10 {
						require.NoError(t, errors.Errorf("torrent did not show up in time"))
					}
					tries++
				}
				require.NotEmpty(t, torrents)
				require.Len(t, torrents, 1)
				require.Equal(t, "B185765BE59BC56AB29A4860A5820863BF0512AD", torrents[0].Hash)
				require.Equal(t, "ubuntu-20.04.2-live-server-arm64.iso", torrents[0].Name)
				require.Equal(t, label.Value, torrents[0].Label)
				require.Equal(t, 1193754624, torrents[0].Size)
				//no path yet since the torrent is stopped
				require.Equal(t, "", torrents[0].Path)
				require.False(t, torrents[0].Completed)

				t.Run("get status", func(t *testing.T) {
					<-time.After(time.Second)
					status, err := client.GetStatus(torrents[0])
					require.NoError(t, err)
					t.Logf("Status = %+v", status)

					require.False(t, status.Completed)
					require.Zero(t, status.CompletedBytes)
					require.Zero(t, status.DownRate)
					require.NotZero(t, status.Size)
				})

				t.Run("delete torrent", func(t *testing.T) {
					err := client.Delete(torrents[0])
					require.NoError(t, err)

					torrents, err := client.GetTorrents(ViewMain)
					require.NoError(t, err)
					require.Empty(t, torrents)

					t.Run("get torrent", func(t *testing.T) {
						// It will take some time to disappear, so retry a few times
						tries := 0
						var torrents []Torrent
						var err error
						for {
							<-time.After(time.Second)
							torrents, err = client.GetTorrents(ViewMain)
							require.NoError(t, err)
							if len(torrents) == 0 {
								break
							}
							if tries > 10 {
								require.NoError(t, errors.Errorf("torrent did not delete in time"))
							}
							tries++
						}
						require.Empty(t, torrents)
					})
				})
			})
		})

		t.Run("with data", func(t *testing.T) {
			b, err := ioutil.ReadFile("testdata/ubuntu-20.04.2-live-server-arm64.iso.torrent")
			require.NoError(t, err)
			require.NotEmpty(t, b)

			err = client.AddTorrent(b)
			require.NoError(t, err)

			t.Run("get torrent", func(t *testing.T) {
				// It will take some time to appear, so retry a few times
				tries := 0
				var torrents []Torrent
				var err error
				for {
					<-time.After(time.Second)
					torrents, err = client.GetTorrents(ViewMain)
					require.NoError(t, err)
					if len(torrents) > 0 {
						break
					}
					if tries > 10 {
						require.NoError(t, errors.Errorf("torrent did not show up in time"))
					}
					tries++
				}
				require.NotEmpty(t, torrents)
				require.Len(t, torrents, 1)
				require.Equal(t, "B185765BE59BC56AB29A4860A5820863BF0512AD", torrents[0].Hash)
				require.Equal(t, "ubuntu-20.04.2-live-server-arm64.iso", torrents[0].Name)
				require.Equal(t, "", torrents[0].Label)
				require.Equal(t, 1193754624, torrents[0].Size)
				require.Equal(t, "/downloads/incoming/ubuntu-20.04.2-live-server-arm64.iso", torrents[0].Path)
				require.False(t, torrents[0].Completed)

				t.Run("get files", func(t *testing.T) {
					files, err := client.GetFiles(torrents[0])
					require.NoError(t, err)
					require.NotEmpty(t, files)
					require.Len(t, files, 1)
					for _, f := range files {
						require.NotEmpty(t, f.Path)
						require.NotZero(t, f.Size)
					}
				})

				t.Run("delete torrent", func(t *testing.T) {
					err := client.Delete(torrents[0])
					require.NoError(t, err)

					torrents, err := client.GetTorrents(ViewMain)
					require.NoError(t, err)
					require.Empty(t, torrents)

					t.Run("get torrent", func(t *testing.T) {
						// It will take some time to disappear, so retry a few times
						tries := 0
						var torrents []Torrent
						var err error
						for {
							<-time.After(time.Second)
							torrents, err = client.GetTorrents(ViewMain)
							require.NoError(t, err)
							if len(torrents) == 0 {
								break
							}
							if tries > 10 {
								require.NoError(t, errors.Errorf("torrent did not delete in time"))
							}
							tries++
						}
						require.Empty(t, torrents)
					})
				})
			})
		})

		t.Run("with data (stopped)", func(t *testing.T) {
			b, err := ioutil.ReadFile("testdata/ubuntu-20.04.2-live-server-arm64.iso.torrent")
			require.NoError(t, err)
			require.NotEmpty(t, b)

			label := DLabel.SetValue("test-label")
			err = client.AddTorrentStopped(b, label)
			require.NoError(t, err)

			t.Run("get torrent", func(t *testing.T) {
				// It will take some time to appear, so retry a few times
				<-time.After(time.Second)
				torrents, err := client.GetTorrents(ViewMain)
				require.NoError(t, err)

				require.NotEmpty(t, torrents)
				require.Len(t, torrents, 1)
				require.Equal(t, "B185765BE59BC56AB29A4860A5820863BF0512AD", torrents[0].Hash)
				require.Equal(t, "ubuntu-20.04.2-live-server-arm64.iso", torrents[0].Name)
				require.Equal(t, label.Value, torrents[0].Label)
				require.Equal(t, 1193754624, torrents[0].Size)

				t.Run("delete torrent", func(t *testing.T) {
					err := client.Delete(torrents[0])
					require.NoError(t, err)

					torrents, err := client.GetTorrents(ViewMain)
					require.NoError(t, err)
					require.Empty(t, torrents)

					t.Run("get torrent", func(t *testing.T) {
						// It will take some time to disappear, so retry a few times
						tries := 0
						var torrents []Torrent
						var err error
						for {
							<-time.After(time.Second)
							torrents, err = client.GetTorrents(ViewMain)
							require.NoError(t, err)
							if len(torrents) == 0 {
								break
							}
							if tries > 10 {
								require.NoError(t, errors.Errorf("torrent did not delete in time"))
							}
							tries++
						}
						require.Empty(t, torrents)
					})
				})
			})
		})
	})

	t.Run("down total post activity", func(t *testing.T) {
		total, err := client.DownTotal()
		require.NoError(t, err)
		require.NotZero(t, total, "expected data to be transferred")
	})

	t.Run("up total post activity", func(t *testing.T) {
		total, err := client.UpTotal()
		require.NoError(t, err)
		require.NotZero(t, total, "expected data to be transferred")
	})
}
