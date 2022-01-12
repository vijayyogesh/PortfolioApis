package auth

import (
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/vijayyogesh/PortfolioApis/constants"
	"github.com/vijayyogesh/PortfolioApis/util"
)

type UserAuth struct{
	UserId string
	Token string
	IsAuthenticated bool
}

/* Fetch new JWT after user enters correct credentials */
func GetJWT(userid string) (string, error) {

	/* Fetch JWT signing key from app env */
	mySigningKey := []byte(util.GetAppUtil().Config.AuthKey)

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	claims["authorized"] = true
	claims["client"] = userid
	claims["aud"] = constants.AppJWTAudience
	claims["iss"] = constants.AppJWTIssuer
	/* Expiry in Hrs set in config */
	claims["exp"] = time.Now().Add(time.Hour * time.Duration(util.GetAppUtil().Config.AuthExp)).Unix()

	tokenString, err := token.SignedString(mySigningKey)

	if err != nil {
		util.GetAppUtil().AppLogger.Println(err)
		return "", err
	}

	return tokenString, nil
}

/* Authenticate Token for subsequent requests */
func AuthenticateToken(r *http.Request, userid string) bool {
	if r.Header["Token"] != nil {

		mySigningKey := []byte(util.GetAppUtil().Config.AuthKey)
		token, err := jwt.Parse(r.Header["Token"][0], func(token *jwt.Token) (interface{}, error) {
			return mySigningKey, nil
		})

		if err != nil {
			util.GetAppUtil().AppLogger.Println("Error while parsing Token")
			util.GetAppUtil().AppLogger.Println(err)
			return false
		}

		/* When Token is valid - compare userid from token and request */
		if token.Valid {
			claims, ok := token.Claims.(jwt.MapClaims)
			if ok {
				util.GetAppUtil().AppLogger.Println("userid in token - ", claims["client"])
				util.GetAppUtil().AppLogger.Println("userid in request - ", userid)

				if userid == claims["client"] {
					util.GetAppUtil().AppLogger.Println("Token Authenticated")
					return true
				} else {
					util.GetAppUtil().AppLogger.Println("Userid in token does not match with User id in request")
					return false
				}
			} else {
				return false
			}
		} else {
			util.GetAppUtil().AppLogger.Println("Invalid Token")
			return false
		}
	} else {
		util.GetAppUtil().AppLogger.Println("Token Not Found")
		return false
	}
}

func GenerateSHA(password string) string {
	hash := sha512.New()
	hash.Write([]byte(password))
	return hex.EncodeToString(hash.Sum(nil))
}
